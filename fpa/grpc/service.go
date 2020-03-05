package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/fil-tools/fpa"
	"github.com/textileio/fil-tools/fpa/fastapi"
	"github.com/textileio/fil-tools/fpa/manager"
	pb "github.com/textileio/fil-tools/fpa/pb"
)

var (
	ErrEmptyAuthToken = errors.New("auth token can't be empty")

	log = logger.Logger("fpa-grpc-service")
)

type Service struct {
	pb.UnimplementedAPIServer

	m   *manager.Manager
	hot fpa.HotLayer
}

func NewService(m *manager.Manager, hot fpa.HotLayer) *Service {
	return &Service{
		m:   m,
		hot: hot,
	}
}

func (s *Service) Info(ctx context.Context, req *pb.InfoRequest) (*pb.InfoReply, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	info, err := i.Info(ctx)
	if err != nil {
		return nil, err
	}

	reply := &pb.InfoReply{
		Id:   info.ID.String(),
		Pins: make([]string, len(info.Pins)),
		Wallet: &pb.WalletInfo{
			Address: info.Wallet.Address,
			Balance: info.Wallet.Balance,
		},
	}
	for i, p := range info.Pins {
		reply.Pins[i] = p.String()
	}

	return reply, nil
}

func (s *Service) Show(ctx context.Context, req *pb.ShowRequest) (*pb.ShowReply, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c, err := cid.Decode(req.GetCid())
	if err != nil {
		return nil, err
	}

	info, err := i.Show(c)
	if err != nil {
		return nil, err
	}
	reply := &pb.ShowReply{
		Cid:     info.Cid.String(),
		Created: info.Created.UnixNano(),
		Hot: &pb.ShowReply_HotInfo{
			Size: int64(info.Hot.Size),
			Ipfs: &pb.ShowReply_IpfsHotInfo{
				Created: info.Hot.Ipfs.Created.UnixNano(),
			},
		},
		Cold: &pb.ShowReply_ColdInfo{
			Filecoin: &pb.ShowReply_FilInfo{
				PayloadCid: info.Cold.Filecoin.PayloadCID.String(),
				Duration:   info.Cold.Filecoin.Duration,
				Proposals:  make([]*pb.ShowReply_FilStorage, len(info.Cold.Filecoin.Proposals)),
			},
		},
	}
	for i, p := range info.Cold.Filecoin.Proposals {
		reply.Cold.Filecoin.Proposals[i] = &pb.ShowReply_FilStorage{
			ProposalCid: p.ProposalCid.String(),
			Failed:      p.Failed,
		}
	}

	return reply, nil
}

func (s *Service) AddCid(ctx context.Context, req *pb.AddCidRequest) (*pb.AddCidReply, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	c, err := cid.Decode(req.GetCid())
	if err != nil {
		return nil, err
	}
	log.Infof("adding cid %s", c)
	jid, err := i.AddCid(c)
	if err != nil {
		return nil, err
	}

	ch := i.Watch(jid)
	defer i.Unwatch(ch)
	for job := range ch {
		if job.Status == fpa.Done {
			break
		} else if job.Status != fpa.Queued && job.Status != fpa.Done {
			return nil, fmt.Errorf("error adding cid: %s", job.ErrCause)
		}
	}
	return &pb.AddCidReply{}, nil
}

func receiveFile(srv pb.API_AddFileServer, writer *io.PipeWriter) {
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
			writer.CloseWithError(writeErr)
		}
	}
}

func (s *Service) AddFile(srv pb.API_AddFileServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}

	reader, writer := io.Pipe()
	defer reader.Close()

	go receiveFile(srv, writer)

	c, err := s.hot.Add(srv.Context(), reader)
	if err != nil {
		return fmt.Errorf("adding data to hot layer: %s", err)
	}

	jid, err := i.AddCid(c)
	if err != nil {
		return err
	}
	ch := i.Watch(jid)
	for job := range ch {
		if job.Status == fpa.Done {
			break
		} else if job.Status == fpa.Failed {
			return fmt.Errorf("error adding cid: %s", job.ErrCause)
		}
	}

	return srv.SendAndClose(&pb.AddFileReply{Cid: c.String()})
}

func (s *Service) Get(req *pb.GetRequest, srv pb.API_GetServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}
	c, err := cid.Decode(req.GetCid())
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
		if sendErr := srv.Send(&pb.GetReply{Chunk: buffer[:bytesRead]}); sendErr != nil {
			return sendErr
		}
		if err == io.EOF {
			return nil
		}
	}
}

func (s *Service) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateReply, error) {
	id, addr, err := s.m.Create(ctx)
	if err != nil {
		log.Errorf("creating instance: %s", err)
		return nil, err
	}
	return &pb.CreateReply{
		Id:      id.String(),
		Address: addr,
	}, nil
}

func (s *Service) getInstanceByToken(ctx context.Context) (*fastapi.Instance, error) {
	token := metautils.ExtractIncoming(ctx).Get("X-fpa-Token")
	if token == "" {
		return nil, ErrEmptyAuthToken
	}
	i, err := s.m.GetByAuthToken(token)
	if err != nil {
		return nil, err
	}
	return i, nil
}
