package powergate

import (
	"context"
	"fmt"
	"io"

	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/api"
	proto "github.com/textileio/powergate/proto/powergate/v1"
	"github.com/textileio/powergate/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Stage allows you to temporarily cache data in the Hot layer in preparation for pushing a cid storage config.
func (s *Service) Stage(srv proto.PowergateService_StageServer) error {
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

	return srv.SendAndClose(&proto.StageResponse{Cid: util.CidToString(c)})
}

// ReplaceData calls ffs.Replace.
func (s *Service) ReplaceData(ctx context.Context, req *proto.ReplaceDataRequest) (*proto.ReplaceDataResponse, error) {
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

	return &proto.ReplaceDataResponse{JobId: jid.String()}, nil
}

// Get gets the data for a stored Cid.
func (s *Service) Get(req *proto.GetRequest, srv proto.PowergateService_GetServer) error {
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
		if sendErr := srv.Send(&proto.GetResponse{Chunk: buffer[:bytesRead]}); sendErr != nil {
			return sendErr
		}
		if err == io.EOF {
			return nil
		}
	}
}

// WatchLogs returns a stream of human-readable messages related to executions of a Cid.
// The listener is automatically unsubscribed when the client closes the stream.
func (s *Service) WatchLogs(req *proto.WatchLogsRequest, srv proto.PowergateService_WatchLogsServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}

	opts := []api.GetLogsOption{api.WithHistory(req.History)}
	if req.JobId != ffs.EmptyJobID.String() {
		opts = append(opts, api.WithJidFilter(ffs.JobID(req.JobId)))
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
		reply := &proto.WatchLogsResponse{
			LogEntry: &proto.LogEntry{
				Cid:     util.CidToString(c),
				JobId:   l.Jid.String(),
				Time:    l.Timestamp.Unix(),
				Message: l.Msg,
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

// CidInfo returns information about cids managed by the FFS instance.
func (s *Service) CidInfo(ctx context.Context, req *proto.CidInfoRequest) (*proto.CidInfoResponse, error) {
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
	res := make([]*proto.CidInfo, 0, len(storageConfigs))
	for cid, config := range storageConfigs {
		rpcConfig := ToRPCStorageConfig(config)
		cidInfo := &proto.CidInfo{
			Cid:                       cid.String(),
			LatestPushedStorageConfig: rpcConfig,
		}
		info, err := i.Show(cid)
		if err != nil && err != api.ErrNotFound {
			return nil, status.Errorf(codes.Internal, "getting storage info: %v", err)
		} else if err == nil {
			cidInfo.CurrentStorageInfo = toRPCStorageInfo(info)
		}
		queuedJobs := i.QueuedStorageJobs(cid)
		rpcQueudJobs := make([]*proto.Job, len(queuedJobs))
		for i, job := range queuedJobs {
			rpcJob, err := toRPCJob(job)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			rpcQueudJobs[i] = rpcJob
		}
		cidInfo.QueuedStorageJobs = rpcQueudJobs
		executingJobs := i.ExecutingStorageJobs()
		if len(executingJobs) > 0 {
			rpcJob, err := toRPCJob(executingJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidInfo.ExecutingStorageJob = rpcJob
		}
		finalJobs := i.LatestFinalStorageJobs(cid)
		if len(finalJobs) > 0 {
			rpcJob, err := toRPCJob(finalJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidInfo.LatestFinalStorageJob = rpcJob
		}
		successfulJobs := i.LatestSuccessfulStorageJobs(cid)
		if len(successfulJobs) > 0 {
			rpcJob, err := toRPCJob(successfulJobs[0])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "converting job to rpc job: %v", err)
			}
			cidInfo.LatestSuccessfulStorageJob = rpcJob
		}
		res = append(res, cidInfo)
	}
	return &proto.CidInfoResponse{CidInfos: res}, nil
}

func receiveFile(srv proto.PowergateService_StageServer, writer *io.PipeWriter) {
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
