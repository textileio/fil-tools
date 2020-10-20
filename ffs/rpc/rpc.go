package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/powergate/deals"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/api"
	"github.com/textileio/powergate/ffs/manager"
	"github.com/textileio/powergate/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrEmptyAuthToken is returned when the provided auth-token is unknown.
	ErrEmptyAuthToken = errors.New("auth token can't be empty")

	log = logger.Logger("ffs-grpc-service")
)

// RPC implements the proto service definition of FFS.
type RPC struct {
	UnimplementedRPCServiceServer

	m   *manager.Manager
	wm  ffs.WalletManager
	hot ffs.HotStorage
}

// New creates a new rpc service.
func New(m *manager.Manager, wm ffs.WalletManager, hot ffs.HotStorage) *RPC {
	return &RPC{
		m:   m,
		wm:  wm,
		hot: hot,
	}
}

// ID returns the API instance id.
func (s *RPC) ID(ctx context.Context, req *IDRequest) (*IDResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	id := i.ID()
	return &IDResponse{Id: id.String()}, nil
}

// Addrs calls ffs.Addrs.
func (s *RPC) Addrs(ctx context.Context, req *AddrsRequest) (*AddrsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}
	addrs := i.Addrs()
	res := make([]*AddrInfo, len(addrs))
	for i, addr := range addrs {
		bal, err := s.wm.Balance(ctx, addr.Addr)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "getting address balance: %v", err)
		}
		res[i] = &AddrInfo{
			Name:    addr.Name,
			Addr:    addr.Addr,
			Type:    addr.Type,
			Balance: bal,
		}
	}
	return &AddrsResponse{Addrs: res}, nil
}

// DefaultStorageConfig calls ffs.DefaultStorageConfig.
func (s *RPC) DefaultStorageConfig(ctx context.Context, req *DefaultStorageConfigRequest) (*DefaultStorageConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	conf := i.DefaultStorageConfig()
	return &DefaultStorageConfigResponse{
		DefaultStorageConfig: ToRPCStorageConfig(conf),
	}, nil
}

// SignMessage calls ffs.SignMessage.
func (s *RPC) SignMessage(ctx context.Context, req *SignMessageRequest) (*SignMessageResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	signature, err := i.SignMessage(ctx, req.Addr, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("signing message: %s", err)
	}

	return &SignMessageResponse{Signature: signature}, nil
}

// VerifyMessage calls ffs.VerifyMessage.
func (s *RPC) VerifyMessage(ctx context.Context, req *VerifyMessageRequest) (*VerifyMessageResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := i.VerifyMessage(ctx, req.Addr, req.Msg, req.Signature)
	if err != nil {
		return nil, fmt.Errorf("verifying signature: %s", err)
	}

	return &VerifyMessageResponse{Ok: ok}, nil
}

// NewAddr calls ffs.NewAddr.
func (s *RPC) NewAddr(ctx context.Context, req *NewAddrRequest) (*NewAddrResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	var opts []api.NewAddressOption
	if req.AddressType != "" {
		opts = append(opts, api.WithAddressType(req.AddressType))
	}
	if req.MakeDefault {
		opts = append(opts, api.WithMakeDefault(req.MakeDefault))
	}

	addr, err := i.NewAddr(ctx, req.Name, opts...)
	if err != nil {
		return nil, err
	}
	return &NewAddrResponse{Addr: addr}, nil
}

// SetDefaultStorageConfig sets a new config to be used by default.
func (s *RPC) SetDefaultStorageConfig(ctx context.Context, req *SetDefaultStorageConfigRequest) (*SetDefaultStorageConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	defaultConfig := ffs.StorageConfig{
		Repairable: req.Config.Repairable,
		Hot:        fromRPCHotConfig(req.Config.Hot),
		Cold:       fromRPCColdConfig(req.Config.Cold),
	}
	if err := i.SetDefaultStorageConfig(defaultConfig); err != nil {
		return nil, err
	}
	return &SetDefaultStorageConfigResponse{}, nil
}

// CidData returns information about cids managed by the FFS instance.
func (s *RPC) CidData(ctx context.Context, req *CidDataRequest) (*CidDataResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}

	storageConfigs, err := i.GetStorageConfigs(cids...)
	if err != nil {
		code := codes.Internal
		if err == api.ErrNotFound {
			code = codes.NotFound
		}
		return nil, status.Errorf(code, "getting storage configs: %v", err)
	}
	res := make([]*CidData, 0, len(storageConfigs))
	for cid, config := range storageConfigs {
		rpcConfig := ToRPCStorageConfig(config)
		cidData := &CidData{
			Cid:                       cid.String(),
			LatestPushedStorageConfig: rpcConfig,
		}
		info, err := i.Show(cid)
		if err != nil && err != api.ErrNotFound {
			return nil, status.Errorf(codes.Internal, "getting storage info: %v", err)
		} else if err == nil {
			cidData.CurrentCidInfo = toRPCCidInfo(info)
		}
		queuedJobs := i.QueuedStorageJobs(cid)
		rpcQueudJobs := make([]*Job, len(queuedJobs))
		for i, job := range queuedJobs {
			rpcJob, err := toRPCJob(job)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			rpcQueudJobs[i] = rpcJob
		}
		cidData.QueuedStorageJobs = rpcQueudJobs
		executingJobs := i.ExecutingStorageJobs()
		if len(executingJobs) > 0 {
			rpcJob, err := toRPCJob(executingJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidData.ExecutingStorageJob = rpcJob
		}
		finalJobs := i.LatestFinalStorageJobs(cid)
		if len(finalJobs) > 0 {
			rpcJob, err := toRPCJob(finalJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidData.LatestFinalStorageJob = rpcJob
		}
		successfulJobs := i.LatestSuccessfulStorageJobs(cid)
		if len(successfulJobs) > 0 {
			rpcJob, err := toRPCJob(successfulJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidData.LatestSuccessfulStorageJob = rpcJob
		}
		res = append(res, cidData)
	}
	return &CidDataResponse{CidDatas: res}, nil
}

// CancelJob calls API.CancelJob.
func (s *RPC) CancelJob(ctx context.Context, req *CancelJobRequest) (*CancelJobResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	jid := ffs.JobID(req.Jid)
	if err := i.CancelJob(jid); err != nil {
		return &CancelJobResponse{}, err
	}
	return &CancelJobResponse{}, nil
}

// StorageJob calls API.GetStorageJob.
func (s *RPC) StorageJob(ctx context.Context, req *StorageJobRequest) (*StorageJobResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	jid := ffs.JobID(req.Jid)
	job, err := i.GetStorageJob(jid)
	if err != nil {
		return nil, err
	}
	rpcJob, err := toRPCJob(job)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "building job response: %v", err.Error())
	}
	return &StorageJobResponse{
		Job: rpcJob,
	}, nil
}

// QueuedStorageJobs returns a list of queued storage jobs.
func (s *RPC) QueuedStorageJobs(ctx context.Context, req *QueuedStorageJobsRequest) (*QueuedStorageJobsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}
	jobs := i.QueuedStorageJobs(cids...)
	protoJobs, err := ToProtoStorageJobs(jobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting jobs to protos: %v", err)
	}
	return &QueuedStorageJobsResponse{
		StorageJobs: protoJobs,
	}, nil
}

// ExecutingStorageJobs returns a list of executing storage jobs.
func (s *RPC) ExecutingStorageJobs(ctx context.Context, req *ExecutingStorageJobsRequest) (*ExecutingStorageJobsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}
	jobs := i.ExecutingStorageJobs(cids...)
	protoJobs, err := ToProtoStorageJobs(jobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting jobs to protos: %v", err)
	}
	return &ExecutingStorageJobsResponse{
		StorageJobs: protoJobs,
	}, nil
}

// LatestFinalStorageJobs returns a list of latest final storage jobs.
func (s *RPC) LatestFinalStorageJobs(ctx context.Context, req *LatestFinalStorageJobsRequest) (*LatestFinalStorageJobsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}
	jobs := i.LatestFinalStorageJobs(cids...)
	protoJobs, err := ToProtoStorageJobs(jobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting jobs to protos: %v", err)
	}
	return &LatestFinalStorageJobsResponse{
		StorageJobs: protoJobs,
	}, nil
}

// LatestSuccessfulStorageJobs returns a list of latest successful storage jobs.
func (s *RPC) LatestSuccessfulStorageJobs(ctx context.Context, req *LatestSuccessfulStorageJobsRequest) (*LatestSuccessfulStorageJobsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}
	jobs := i.LatestSuccessfulStorageJobs(cids...)
	protoJobs, err := ToProtoStorageJobs(jobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting jobs to protos: %v", err)
	}
	return &LatestSuccessfulStorageJobsResponse{
		StorageJobs: protoJobs,
	}, nil
}

// StorageJobsSummary returns a summary of all storage jobs.
func (s *RPC) StorageJobsSummary(ctx context.Context, req *StorageJobsSummaryRequest) (*StorageJobsSummaryResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "getting instance: %v", err)
	}

	cids, err := fromProtoCids(req.Cids)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing cids: %v", err)
	}

	queuedJobs := i.QueuedStorageJobs(cids...)
	executingJobs := i.ExecutingStorageJobs(cids...)
	latestFinalJobs := i.LatestFinalStorageJobs(cids...)
	latestSuccessfulJobs := i.LatestSuccessfulStorageJobs(cids...)

	protoQueuedJobs, err := ToProtoStorageJobs(queuedJobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting queued jobs to protos: %v", err)
	}
	protoExecutingJobs, err := ToProtoStorageJobs(executingJobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting executing jobs to protos: %v", err)
	}
	protoLatestFinalJobs, err := ToProtoStorageJobs(latestFinalJobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting latest final jobs to protos: %v", err)
	}
	protoLatestSuccessfulJobs, err := ToProtoStorageJobs(latestSuccessfulJobs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting latest successful jobs to protos: %v", err)
	}

	return &StorageJobsSummaryResponse{
		JobCounts: &JobCounts{
			Executing:        int32(len(executingJobs)),
			LatestFinal:      int32(len(latestFinalJobs)),
			LatestSuccessful: int32(len(latestSuccessfulJobs)),
			Queued:           int32(len(queuedJobs)),
		},
		ExecutingStorageJobs:        protoExecutingJobs,
		LatestFinalStorageJobs:      protoLatestFinalJobs,
		LatestSuccessfulStorageJobs: protoLatestSuccessfulJobs,
		QueuedStorageJobs:           protoQueuedJobs,
	}, nil
}

// WatchJobs calls API.WatchJobs.
func (s *RPC) WatchJobs(req *WatchJobsRequest, srv RPCService_WatchJobsServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}

	jids := make([]ffs.JobID, len(req.Jids))
	for i, jid := range req.Jids {
		jids[i] = ffs.JobID(jid)
	}

	ch := make(chan ffs.StorageJob, 100)
	go func() {
		err = i.WatchJobs(srv.Context(), ch, jids...)
		close(ch)
	}()
	for job := range ch {
		rpcJob, err := toRPCJob(job)
		if err != nil {
			return err
		}
		reply := &WatchJobsResponse{
			Job: rpcJob,
		}
		if err := srv.Send(reply); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return nil
}

// WatchLogs returns a stream of human-readable messages related to executions of a Cid.
// The listener is automatically unsubscribed when the client closes the stream.
func (s *RPC) WatchLogs(req *WatchLogsRequest, srv RPCService_WatchLogsServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}

	opts := []api.GetLogsOption{api.WithHistory(req.History)}
	if req.Jid != ffs.EmptyJobID.String() {
		opts = append(opts, api.WithJidFilter(ffs.JobID(req.Jid)))
	}

	c, err := util.CidFromString(req.Cid)
	if err != nil {
		return err
	}
	ch := make(chan ffs.LogEntry, 100)
	go func() {
		err = i.WatchLogs(srv.Context(), ch, c, opts...)
		close(ch)
	}()
	for l := range ch {
		reply := &WatchLogsResponse{
			LogEntry: &LogEntry{
				Cid:  util.CidToString(c),
				Jid:  l.Jid.String(),
				Time: l.Timestamp.Unix(),
				Msg:  l.Msg,
			},
		}
		if err := srv.Send(reply); err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}

	return nil
}

// Replace calls ffs.Replace.
func (s *RPC) Replace(ctx context.Context, req *ReplaceRequest) (*ReplaceResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c1, err := util.CidFromString(req.Cid1)
	if err != nil {
		return nil, err
	}
	c2, err := util.CidFromString(req.Cid2)
	if err != nil {
		return nil, err
	}

	jid, err := i.Replace(c1, c2)
	if err != nil {
		return nil, err
	}

	return &ReplaceResponse{JobId: jid.String()}, nil
}

// PushStorageConfig applies the provided cid storage config.
func (s *RPC) PushStorageConfig(ctx context.Context, req *PushStorageConfigRequest) (*PushStorageConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c, err := util.CidFromString(req.Cid)
	if err != nil {
		return nil, err
	}

	options := []api.PushStorageConfigOption{}

	if req.HasConfig {
		config := ffs.StorageConfig{
			Repairable: req.Config.Repairable,
			Hot:        fromRPCHotConfig(req.Config.Hot),
			Cold:       fromRPCColdConfig(req.Config.Cold),
		}
		options = append(options, api.WithStorageConfig(config))
	}

	if req.HasOverrideConfig {
		options = append(options, api.WithOverride(req.OverrideConfig))
	}

	jid, err := i.PushStorageConfig(c, options...)
	if err != nil {
		return nil, err
	}

	return &PushStorageConfigResponse{
		JobId: jid.String(),
	}, nil
}

// Remove calls ffs.Remove.
func (s *RPC) Remove(ctx context.Context, req *RemoveRequest) (*RemoveResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c, err := util.CidFromString(req.Cid)
	if err != nil {
		return nil, err
	}

	if err := i.Remove(c); err != nil {
		return nil, err
	}

	return &RemoveResponse{}, nil
}

// Get gets the data for a stored Cid.
func (s *RPC) Get(req *GetRequest, srv RPCService_GetServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}
	c, err := util.CidFromString(req.GetCid())
	if err != nil {
		return err
	}
	r, err := i.Get(srv.Context(), c)
	if err != nil {
		return err
	}

	buffer := make([]byte, 1024*32)
	for {
		bytesRead, err := r.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if sendErr := srv.Send(&GetResponse{Chunk: buffer[:bytesRead]}); sendErr != nil {
			return sendErr
		}
		if err == io.EOF {
			return nil
		}
	}
}

// SendFil sends fil from a managed address to any other address.
func (s *RPC) SendFil(ctx context.Context, req *SendFilRequest) (*SendFilResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	if err := i.SendFil(ctx, req.From, req.To, big.NewInt(req.Amount)); err != nil {
		return nil, err
	}
	return &SendFilResponse{}, nil
}

// Stage allows you to temporarily cache data in the Hot layer in preparation for pushing a cid storage config.
func (s *RPC) Stage(srv RPCService_StageServer) error {
	// check that an API instance exists so not just anyone can add data to the hot layer
	if _, err := s.getInstanceByToken(srv.Context()); err != nil {
		return err
	}

	reader, writer := io.Pipe()
	defer func() {
		if err := reader.Close(); err != nil {
			log.Errorf("closing reader: %s", err)
		}
	}()

	go receiveFile(srv, writer)

	c, err := s.hot.Add(srv.Context(), reader)
	if err != nil {
		return fmt.Errorf("adding data to hot storage: %s", err)
	}

	return srv.SendAndClose(&StageResponse{Cid: util.CidToString(c)})
}

// ListPayChannels lists all pay channels.
func (s *RPC) ListPayChannels(ctx context.Context, req *ListPayChannelsRequest) (*ListPayChannelsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	infos, err := i.ListPayChannels(ctx)
	if err != nil {
		return nil, err
	}
	respInfos := make([]*PaychInfo, len(infos))
	for i, info := range infos {
		respInfos[i] = toRPCPaychInfo(info)
	}
	return &ListPayChannelsResponse{PayChannels: respInfos}, nil
}

// CreatePayChannel creates a payment channel.
func (s *RPC) CreatePayChannel(ctx context.Context, req *CreatePayChannelRequest) (*CreatePayChannelResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	info, cid, err := i.CreatePayChannel(ctx, req.From, req.To, req.Amount)
	if err != nil {
		return nil, err
	}
	respInfo := toRPCPaychInfo(info)
	return &CreatePayChannelResponse{
		PayChannel:        respInfo,
		ChannelMessageCid: util.CidToString(cid),
	}, nil
}

// RedeemPayChannel redeems a payment channel.
func (s *RPC) RedeemPayChannel(ctx context.Context, req *RedeemPayChannelRequest) (*RedeemPayChannelResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	if err := i.RedeemPayChannel(ctx, req.PayChannelAddr); err != nil {
		return nil, err
	}
	return &RedeemPayChannelResponse{}, nil
}

// ListStorageDealRecords calls ffs.ListStorageDealRecords.
func (s *RPC) ListStorageDealRecords(ctx context.Context, req *ListStorageDealRecordsRequest) (*ListStorageDealRecordsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	records, err := i.ListStorageDealRecords(buildListDealRecordsOptions(req.Config)...)
	if err != nil {
		return nil, err
	}
	return &ListStorageDealRecordsResponse{Records: toRPCStorageDealRecords(records)}, nil
}

// ListRetrievalDealRecords calls ffs.ListRetrievalDealRecords.
func (s *RPC) ListRetrievalDealRecords(ctx context.Context, req *ListRetrievalDealRecordsRequest) (*ListRetrievalDealRecordsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	records, err := i.ListRetrievalDealRecords(buildListDealRecordsOptions(req.Config)...)
	if err != nil {
		return nil, err
	}
	return &ListRetrievalDealRecordsResponse{Records: toRPCRetrievalDealRecords(records)}, nil
}

func (s *RPC) getInstanceByToken(ctx context.Context) (*api.API, error) {
	token := metautils.ExtractIncoming(ctx).Get("X-ffs-Token")
	if token == "" {
		return nil, ErrEmptyAuthToken
	}
	i, err := s.m.GetByAuthToken(token)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func receiveFile(srv RPCService_StageServer, writer *io.PipeWriter) {
	for {
		req, err := srv.Recv()
		if err == io.EOF {
			_ = writer.Close()
			break
		} else if err != nil {
			_ = writer.CloseWithError(err)
			break
		}
		_, writeErr := writer.Write(req.GetChunk())
		if writeErr != nil {
			if err := writer.CloseWithError(writeErr); err != nil {
				log.Errorf("closing with error: %s", err)
			}
		}
	}
}

// ToRPCStorageConfig converts from a ffs.StorageConfig to a rpc StorageConfig.
func ToRPCStorageConfig(config ffs.StorageConfig) *StorageConfig {
	return &StorageConfig{
		Repairable: config.Repairable,
		Hot:        toRPCHotConfig(config.Hot),
		Cold:       toRPCColdConfig(config.Cold),
	}
}

func toRPCHotConfig(config ffs.HotConfig) *HotConfig {
	return &HotConfig{
		Enabled:          config.Enabled,
		AllowUnfreeze:    config.AllowUnfreeze,
		UnfreezeMaxPrice: config.UnfreezeMaxPrice,
		Ipfs: &IpfsConfig{
			AddTimeout: int64(config.Ipfs.AddTimeout),
		},
	}
}

func toRPCColdConfig(config ffs.ColdConfig) *ColdConfig {
	return &ColdConfig{
		Enabled: config.Enabled,
		Filecoin: &FilConfig{
			RepFactor:       int64(config.Filecoin.RepFactor),
			DealMinDuration: config.Filecoin.DealMinDuration,
			ExcludedMiners:  config.Filecoin.ExcludedMiners,
			TrustedMiners:   config.Filecoin.TrustedMiners,
			CountryCodes:    config.Filecoin.CountryCodes,
			Renew: &FilRenew{
				Enabled:   config.Filecoin.Renew.Enabled,
				Threshold: int64(config.Filecoin.Renew.Threshold),
			},
			Addr:            config.Filecoin.Addr,
			MaxPrice:        config.Filecoin.MaxPrice,
			FastRetrieval:   config.Filecoin.FastRetrieval,
			DealStartOffset: config.Filecoin.DealStartOffset,
		},
	}
}

func toRPCDealErrors(des []ffs.DealError) []*DealError {
	ret := make([]*DealError, len(des))
	for i, de := range des {
		var strProposalCid string
		if de.ProposalCid.Defined() {
			strProposalCid = util.CidToString(de.ProposalCid)
		}
		ret[i] = &DealError{
			ProposalCid: strProposalCid,
			Miner:       de.Miner,
			Message:     de.Message,
		}
	}
	return ret
}

func fromRPCHotConfig(config *HotConfig) ffs.HotConfig {
	res := ffs.HotConfig{}
	if config != nil {
		res.Enabled = config.Enabled
		res.AllowUnfreeze = config.AllowUnfreeze
		res.UnfreezeMaxPrice = config.UnfreezeMaxPrice
		if config.Ipfs != nil {
			ipfs := ffs.IpfsConfig{
				AddTimeout: int(config.Ipfs.AddTimeout),
			}
			res.Ipfs = ipfs
		}
	}
	return res
}

func fromRPCColdConfig(config *ColdConfig) ffs.ColdConfig {
	res := ffs.ColdConfig{}
	if config != nil {
		res.Enabled = config.Enabled
		if config.Filecoin != nil {
			filecoin := ffs.FilConfig{
				RepFactor:       int(config.Filecoin.RepFactor),
				DealMinDuration: config.Filecoin.DealMinDuration,
				ExcludedMiners:  config.Filecoin.ExcludedMiners,
				CountryCodes:    config.Filecoin.CountryCodes,
				TrustedMiners:   config.Filecoin.TrustedMiners,
				Addr:            config.Filecoin.Addr,
				MaxPrice:        config.Filecoin.MaxPrice,
				FastRetrieval:   config.Filecoin.FastRetrieval,
				DealStartOffset: config.Filecoin.DealStartOffset,
			}
			if config.Filecoin.Renew != nil {
				renew := ffs.FilRenew{
					Enabled:   config.Filecoin.Renew.Enabled,
					Threshold: int(config.Filecoin.Renew.Threshold),
				}
				filecoin.Renew = renew
			}
			res.Filecoin = filecoin
		}
	}
	return res
}

func toRPCCidInfo(info ffs.CidInfo) *CidInfo {
	cidInfo := &CidInfo{
		JobId:   info.JobID.String(),
		Cid:     util.CidToString(info.Cid),
		Created: info.Created.UnixNano(),
		Hot: &HotInfo{
			Enabled: info.Hot.Enabled,
			Size:    int64(info.Hot.Size),
			Ipfs: &IpfsHotInfo{
				Created: info.Hot.Ipfs.Created.UnixNano(),
			},
		},
		Cold: &ColdInfo{
			Enabled: info.Cold.Enabled,
			Filecoin: &FilInfo{
				DataCid:   util.CidToString(info.Cold.Filecoin.DataCid),
				Size:      info.Cold.Filecoin.Size,
				Proposals: make([]*FilStorage, len(info.Cold.Filecoin.Proposals)),
			},
		},
	}
	for i, p := range info.Cold.Filecoin.Proposals {
		var strProposalCid string
		if p.ProposalCid.Defined() {
			strProposalCid = util.CidToString(p.ProposalCid)
		}
		var strPieceCid string
		if p.PieceCid.Defined() {
			strPieceCid = util.CidToString(p.PieceCid)
		}
		cidInfo.Cold.Filecoin.Proposals[i] = &FilStorage{
			ProposalCid:     strProposalCid,
			PieceCid:        strPieceCid,
			Renewed:         p.Renewed,
			Duration:        p.Duration,
			ActivationEpoch: p.ActivationEpoch,
			StartEpoch:      p.StartEpoch,
			Miner:           p.Miner,
			EpochPrice:      p.EpochPrice,
		}
	}
	return cidInfo
}

func toRPCPaychInfo(info ffs.PaychInfo) *PaychInfo {
	var direction Direction
	switch info.Direction {
	case ffs.PaychDirInbound:
		direction = Direction_DIRECTION_INBOUND
	case ffs.PaychDirOutbound:
		direction = Direction_DIRECTION_OUTBOUND
	default:
		direction = Direction_DIRECTION_UNSPECIFIED
	}
	return &PaychInfo{
		CtlAddr:   info.CtlAddr,
		Addr:      info.Addr,
		Direction: direction,
	}
}

func buildListDealRecordsOptions(conf *ListDealRecordsConfig) []deals.ListDealRecordsOption {
	var opts []deals.ListDealRecordsOption
	if conf != nil {
		opts = []deals.ListDealRecordsOption{
			deals.WithAscending(conf.Ascending),
			deals.WithDataCids(conf.DataCids...),
			deals.WithFromAddrs(conf.FromAddrs...),
			deals.WithIncludePending(conf.IncludePending),
			deals.WithIncludeFinal(conf.IncludeFinal),
		}
	}
	return opts
}

func toRPCStorageDealRecords(records []deals.StorageDealRecord) []*StorageDealRecord {
	ret := make([]*StorageDealRecord, len(records))
	for i, r := range records {
		ret[i] = &StorageDealRecord{
			RootCid: util.CidToString(r.RootCid),
			Addr:    r.Addr,
			Time:    r.Time,
			Pending: r.Pending,
			DealInfo: &StorageDealInfo{
				ProposalCid:     util.CidToString(r.DealInfo.ProposalCid),
				StateId:         r.DealInfo.StateID,
				StateName:       r.DealInfo.StateName,
				Miner:           r.DealInfo.Miner,
				PieceCid:        util.CidToString(r.DealInfo.PieceCID),
				Size:            r.DealInfo.Size,
				PricePerEpoch:   r.DealInfo.PricePerEpoch,
				StartEpoch:      r.DealInfo.StartEpoch,
				Duration:        r.DealInfo.Duration,
				DealId:          r.DealInfo.DealID,
				ActivationEpoch: r.DealInfo.ActivationEpoch,
				Msg:             r.DealInfo.Message,
			},
		}
	}
	return ret
}

func toRPCRetrievalDealRecords(records []deals.RetrievalDealRecord) []*RetrievalDealRecord {
	ret := make([]*RetrievalDealRecord, len(records))
	for i, r := range records {
		ret[i] = &RetrievalDealRecord{
			Addr: r.Addr,
			Time: r.Time,
			DealInfo: &RetrievalDealInfo{
				RootCid:                 util.CidToString(r.DealInfo.RootCid),
				Size:                    r.DealInfo.Size,
				MinPrice:                r.DealInfo.MinPrice,
				PaymentInterval:         r.DealInfo.PaymentInterval,
				PaymentIntervalIncrease: r.DealInfo.PaymentIntervalIncrease,
				Miner:                   r.DealInfo.Miner,
				MinerPeerId:             r.DealInfo.MinerPeerID,
			},
		}
	}
	return ret
}

// ToProtoStorageJobs converts a slice of ffs.StorageJobs to proto Jobs.
func ToProtoStorageJobs(jobs []ffs.StorageJob) ([]*Job, error) {
	var res []*Job
	for _, job := range jobs {
		j, err := toRPCJob(job)
		if err != nil {
			return nil, err
		}
		res = append(res, j)
	}
	return res, nil
}

func toRPCJob(job ffs.StorageJob) (*Job, error) {
	var dealInfo []*DealInfo
	for _, item := range job.DealInfo {
		info := &DealInfo{
			ActivationEpoch: item.ActivationEpoch,
			DealId:          item.DealID,
			Duration:        item.Duration,
			Message:         item.Message,
			Miner:           item.Miner,
			PieceCid:        item.PieceCID.String(),
			PricePerEpoch:   item.PricePerEpoch,
			ProposalCid:     item.ProposalCid.String(),
			Size:            item.Size,
			StartEpoch:      item.Size,
			StateId:         item.StateID,
			StateName:       item.StateName,
		}
		dealInfo = append(dealInfo, info)
	}

	var status JobStatus
	switch job.Status {
	case ffs.Unspecified:
		status = JobStatus_JOB_STATUS_UNSPECIFIED
	case ffs.Queued:
		status = JobStatus_JOB_STATUS_QUEUED
	case ffs.Executing:
		status = JobStatus_JOB_STATUS_EXECUTING
	case ffs.Failed:
		status = JobStatus_JOB_STATUS_FAILED
	case ffs.Canceled:
		status = JobStatus_JOB_STATUS_CANCELED
	case ffs.Success:
		status = JobStatus_JOB_STATUS_SUCCESS
	default:
		return nil, fmt.Errorf("unknown job status: %v", job.Status)
	}
	return &Job{
		Id:         job.ID.String(),
		ApiId:      job.APIID.String(),
		Cid:        util.CidToString(job.Cid),
		Status:     status,
		ErrCause:   job.ErrCause,
		DealErrors: toRPCDealErrors(job.DealErrors),
		CreatedAt:  job.CreatedAt,
		DealInfo:   dealInfo,
	}, nil
}

func fromProtoCids(cids []string) ([]cid.Cid, error) {
	var res []cid.Cid
	for _, cid := range cids {
		cid, err := util.CidFromString(cid)
		if err != nil {
			return nil, err
		}
		res = append(res, cid)
	}
	return res, nil
}
