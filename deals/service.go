package deals

import (
	"context"
	"io"

	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	pb "github.com/textileio/fil-tools/deals/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the gprc service
type Service struct {
	pb.UnimplementedAPIServer

	Module *Module
}

type storeResult struct {
	DataCid      cid.Cid
	ProposalCids []cid.Cid
	FailedDeals  []StorageDealConfig
	Err          error
}

// NewService is a helper to create a new Service
func NewService(dm *Module) *Service {
	return &Service{
		Module: dm,
	}
}

func store(ctx context.Context, dealsModule *Module, storeParams *pb.StoreParams, reader io.Reader, ch chan storeResult) {
	defer close(ch)
	dealConfigs := make([]StorageDealConfig, len(storeParams.GetDealConfigs()))
	for i, dealConfig := range storeParams.GetDealConfigs() {
		dealConfigs[i] = StorageDealConfig{
			Miner:      dealConfig.GetMiner(),
			EpochPrice: types.NewInt(dealConfig.GetEpochPrice()),
		}
	}
	dcid, pcids, failedDeals, err := dealsModule.Store(ctx, storeParams.GetAddress(), reader, dealConfigs, storeParams.GetDuration())
	if err != nil {
		ch <- storeResult{Err: err}
		return
	}
	ch <- storeResult{DataCid: dcid, ProposalCids: pcids, FailedDeals: failedDeals}
}

// Store calls deals.Store
func (s *Service) Store(srv pb.API_StoreServer) error {
	req, err := srv.Recv()
	if err != nil {
		return err
	}
	var storeParams *pb.StoreParams
	switch payload := req.GetPayload().(type) {
	case *pb.StoreRequest_StoreParams:
		storeParams = payload.StoreParams
	default:
		return status.Errorf(codes.InvalidArgument, "expected StoreParams for StoreRequest.Payload but got %T", payload)
	}

	reader, writer := io.Pipe()

	storeChannel := make(chan storeResult)
	go store(srv.Context(), s.Module, storeParams, reader, storeChannel)

	for {
		req, err := srv.Recv()
		if err == io.EOF {
			_ = writer.Close()
			break
		} else if err != nil {
			_ = writer.CloseWithError(err)
			break
		}
		switch payload := req.GetPayload().(type) {
		case *pb.StoreRequest_Chunk:
			_, writeErr := writer.Write(payload.Chunk)
			if writeErr != nil {
				return writeErr
			}
		default:
			return status.Errorf(codes.InvalidArgument, "expected Chunk for StoreRequest.Payload but got %T", payload)
		}
	}

	storeResult := <-storeChannel
	if storeResult.Err != nil {
		return storeResult.Err
	}

	replyCids := make([]string, len(storeResult.ProposalCids))
	for i, cid := range storeResult.ProposalCids {
		replyCids[i] = cid.String()
	}

	replyFailedDeals := make([]*pb.DealConfig, len(storeResult.FailedDeals))
	for i, dealConfig := range storeResult.FailedDeals {
		replyFailedDeals[i] = &pb.DealConfig{Miner: dealConfig.Miner, EpochPrice: dealConfig.EpochPrice.Uint64()}
	}

	return srv.SendAndClose(&pb.StoreReply{DataCid: storeResult.DataCid.String(), ProposalCids: replyCids, FailedDeals: replyFailedDeals})
}

// Watch calls deals.Watch
func (s *Service) Watch(req *pb.WatchRequest, srv pb.API_WatchServer) error {
	proposals := make([]cid.Cid, len(req.GetProposals()))
	for i, proposal := range req.GetProposals() {
		id, err := cid.Decode(proposal)
		if err != nil {
			return err
		}
		proposals[i] = id
	}
	ch, err := s.Module.Watch(srv.Context(), proposals)
	if err != nil {
		return err
	}

	for update := range ch {
		dealInfo := &pb.DealInfo{
			ProposalCid:   update.ProposalCid.String(),
			StateID:       update.StateID,
			StateName:     update.StateName,
			Miner:         update.Miner,
			PieceRef:      update.PieceRef,
			Size:          update.Size,
			PricePerEpoch: update.PricePerEpoch.Uint64(),
			Duration:      update.Duration,
		}
		srv.Send(&pb.WatchReply{DealInfo: dealInfo})
	}
	return nil
}
