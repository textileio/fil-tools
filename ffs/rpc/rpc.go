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
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/api"
	"github.com/textileio/powergate/ffs/manager"
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
	hot ffs.HotStorage
}

// New creates a new rpc service
func New(m *manager.Manager, hot ffs.HotStorage) *RPC {
	return &RPC{
		m:   m,
		hot: hot,
	}
}

// Create creates a new Api.
func (s *RPC) Create(ctx context.Context, req *CreateRequest) (*CreateResponse, error) {
	id, token, err := s.m.Create(ctx)
	if err != nil {
		log.Errorf("creating instance: %s", err)
		return nil, err
	}
	return &CreateResponse{
		Id:    id.String(),
		Token: token,
	}, nil
}

// ListAPI returns a list of all existing API instances.
func (s *RPC) ListAPI(ctx context.Context, req *ListAPIRequest) (*ListAPIResponse, error) {
	lst, err := s.m.List()
	if err != nil {
		log.Errorf("listing instances: %s", err)
		return nil, err
	}
	ins := make([]string, len(lst))
	for i, v := range lst {
		ins[i] = v.String()
	}
	return &ListAPIResponse{
		Instances: ins,
	}, nil
}

// ID returns the API instance id
func (s *RPC) ID(ctx context.Context, req *IDRequest) (*IDResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	id := i.ID()
	return &IDResponse{Id: id.String()}, nil
}

// Addrs calls ffs.Addrs
func (s *RPC) Addrs(ctx context.Context, req *AddrsRequest) (*AddrsResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	addrs := i.Addrs()
	res := make([]*AddrInfo, len(addrs))
	for i, addr := range addrs {
		res[i] = &AddrInfo{
			Name: addr.Name,
			Addr: addr.Addr,
			Type: addr.Type,
		}
	}
	return &AddrsResponse{Addrs: res}, nil
}

// DefaultConfig calls ffs.DefaultConfig
func (s *RPC) DefaultConfig(ctx context.Context, req *DefaultConfigRequest) (*DefaultConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	conf := i.DefaultConfig()
	return &DefaultConfigResponse{
		DefaultConfig: &DefaultConfig{
			Hot:        toRPCHotConfig(conf.Hot),
			Cold:       toRPCColdConfig(conf.Cold),
			Repairable: conf.Repairable,
		},
	}, nil
}

// NewAddr calls ffs.NewAddr
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

// GetDefaultCidConfig returns the default cid config prepped for the provided cid
func (s *RPC) GetDefaultCidConfig(ctx context.Context, req *GetDefaultCidConfigRequest) (*GetDefaultCidConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	c, err := cid.Decode(req.Cid)
	if err != nil {
		return nil, err
	}
	config := i.GetDefaultCidConfig(c)
	return &GetDefaultCidConfigResponse{
		Config: &CidConfig{
			Cid:        config.Cid.String(),
			Hot:        toRPCHotConfig(config.Hot),
			Cold:       toRPCColdConfig(config.Cold),
			Repairable: config.Repairable,
		},
	}, nil
}

// GetCidConfig returns the cid config for the provided cid
func (s *RPC) GetCidConfig(ctx context.Context, req *GetCidConfigRequest) (*GetCidConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	c, err := cid.Decode(req.Cid)
	if err != nil {
		return nil, err
	}
	config, err := i.GetCidConfig(c)
	if err != nil {
		return nil, err
	}
	return &GetCidConfigResponse{
		Config: &CidConfig{
			Cid:        config.Cid.String(),
			Hot:        toRPCHotConfig(config.Hot),
			Cold:       toRPCColdConfig(config.Cold),
			Repairable: config.Repairable,
		},
	}, nil
}

// SetDefaultConfig sets a new config to be used by default
func (s *RPC) SetDefaultConfig(ctx context.Context, req *SetDefaultConfigRequest) (*SetDefaultConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	defaultConfig := ffs.DefaultConfig{
		Repairable: req.Config.Repairable,
		Hot:        fromRPCHotConfig(req.Config.Hot),
		Cold:       fromRPCColdConfig(req.Config.Cold),
	}
	if err := i.SetDefaultConfig(defaultConfig); err != nil {
		return nil, err
	}
	return &SetDefaultConfigResponse{}, nil
}

// Show returns information about a particular Cid.
func (s *RPC) Show(ctx context.Context, req *ShowRequest) (*ShowResponse, error) {
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
	reply := &ShowResponse{
		CidInfo: toRPCCidInfo(info),
	}
	return reply, nil
}

// Info returns an Api information.
func (s *RPC) Info(ctx context.Context, req *InfoRequest) (*InfoResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	info, err := i.Info(ctx)
	if err != nil {
		return nil, err
	}

	balances := make([]*BalanceInfo, len(info.Balances))
	for i, balanceInfo := range info.Balances {
		balances[i] = &BalanceInfo{
			Addr: &AddrInfo{
				Name: balanceInfo.Name,
				Addr: balanceInfo.Addr,
				Type: balanceInfo.Type,
			},
			Balance: int64(balanceInfo.Balance),
		}
	}

	reply := &InfoResponse{
		Info: &InstanceInfo{
			Id: info.ID.String(),
			DefaultConfig: &DefaultConfig{
				Hot:        toRPCHotConfig(info.DefaultConfig.Hot),
				Cold:       toRPCColdConfig(info.DefaultConfig.Cold),
				Repairable: info.DefaultConfig.Repairable,
			},
			Balances: balances,
			Pins:     make([]string, len(info.Pins)),
		},
	}
	for i, p := range info.Pins {
		reply.Info.Pins[i] = p.String()
	}
	return reply, nil
}

// WatchJobs calls API.WatchJobs
func (s *RPC) WatchJobs(req *WatchJobsRequest, srv RPCService_WatchJobsServer) error {
	i, err := s.getInstanceByToken(srv.Context())
	if err != nil {
		return err
	}

	jids := make([]ffs.JobID, len(req.Jids))
	for i, jid := range req.Jids {
		jids[i] = ffs.JobID(jid)
	}

	ch := make(chan ffs.Job, 100)
	go func() {
		err = i.WatchJobs(srv.Context(), ch, jids...)
		close(ch)
	}()
	for job := range ch {
		var status JobStatus
		switch job.Status {
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
			status = JobStatus_JOB_STATUS_UNSPECIFIED
		}
		reply := &WatchJobsResponse{
			Job: &Job{
				Id:         job.ID.String(),
				ApiId:      job.APIID.String(),
				Cid:        job.Cid.String(),
				Status:     status,
				ErrCause:   job.ErrCause,
				DealErrors: toRPCDealErrors(job.DealErrors),
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

	c, err := cid.Decode(req.Cid)
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
				Cid:  c.String(),
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

// Replace calls ffs.Replace
func (s *RPC) Replace(ctx context.Context, req *ReplaceRequest) (*ReplaceResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c1, err := cid.Decode(req.Cid1)
	if err != nil {
		return nil, err
	}
	c2, err := cid.Decode(req.Cid2)
	if err != nil {
		return nil, err
	}

	jid, err := i.Replace(c1, c2)
	if err != nil {
		return nil, err
	}

	return &ReplaceResponse{JobId: jid.String()}, nil
}

// PushConfig applies the provided cid config
func (s *RPC) PushConfig(ctx context.Context, req *PushConfigRequest) (*PushConfigResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c, err := cid.Decode(req.Cid)
	if err != nil {
		return nil, err
	}

	options := []api.PushConfigOption{}

	if req.HasConfig {
		cid, err := cid.Decode(req.Config.Cid)
		if err != nil {
			return nil, err
		}
		config := ffs.CidConfig{
			Cid:        cid,
			Repairable: req.Config.Repairable,
			Hot:        fromRPCHotConfig(req.Config.Hot),
			Cold:       fromRPCColdConfig(req.Config.Cold),
		}
		options = append(options, api.WithCidConfig(config))
	}

	if req.HasOverrideConfig {
		options = append(options, api.WithOverride(req.OverrideConfig))
	}

	jid, err := i.PushConfig(c, options...)
	if err != nil {
		return nil, err
	}

	return &PushConfigResponse{
		JobId: jid.String(),
	}, nil
}

// Remove calls ffs.Remove
func (s *RPC) Remove(ctx context.Context, req *RemoveRequest) (*RemoveResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}

	c, err := cid.Decode(req.Cid)
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
		if sendErr := srv.Send(&GetResponse{Chunk: buffer[:bytesRead]}); sendErr != nil {
			return sendErr
		}
		if err == io.EOF {
			return nil
		}
	}
}

// SendFil sends fil from a managed address to any other address
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

// Close calls API.Close
func (s *RPC) Close(ctx context.Context, req *CloseRequest) (*CloseResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	if err := i.Close(); err != nil {
		return nil, err
	}
	return &CloseResponse{}, nil
}

// AddToHot stores data in the Hot Storage so the resulting cid can be used in PushConfig
func (s *RPC) AddToHot(srv RPCService_AddToHotServer) error {
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

	return srv.SendAndClose(&AddToHotResponse{Cid: c.String()})
}

// ListPayChannels lists all pay channels
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

// CreatePayChannel creates a payment channel
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
		ChannelMessageCid: cid.String(),
	}, nil
}

// RedeemPayChannel redeems a payment channel
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

// ShowAll returns a list of CidInfo for all data stored in the FFS instance
func (s *RPC) ShowAll(ctx context.Context, req *ShowAllRequest) (*ShowAllResponse, error) {
	i, err := s.getInstanceByToken(ctx)
	if err != nil {
		return nil, err
	}
	instanceInfo, err := i.Info(ctx)
	if err != nil {
		return nil, err
	}
	cidInfos := make([]*CidInfo, len(instanceInfo.Pins))
	for j, cid := range instanceInfo.Pins {
		cidInfo, err := i.Show(cid)
		if err != nil {
			return nil, err
		}
		cidInfos[j] = toRPCCidInfo(cidInfo)
	}
	return &ShowAllResponse{CidInfos: cidInfos}, nil
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

func receiveFile(srv RPCService_AddToHotServer, writer *io.PipeWriter) {
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

func toRPCHotConfig(config ffs.HotConfig) *HotConfig {
	return &HotConfig{
		Enabled:       config.Enabled,
		AllowUnfreeze: config.AllowUnfreeze,
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
			DealMinDuration: int64(config.Filecoin.DealMinDuration),
			ExcludedMiners:  config.Filecoin.ExcludedMiners,
			TrustedMiners:   config.Filecoin.TrustedMiners,
			CountryCodes:    config.Filecoin.CountryCodes,
			Renew: &FilRenew{
				Enabled:   config.Filecoin.Renew.Enabled,
				Threshold: int64(config.Filecoin.Renew.Threshold),
			},
			Addr:     config.Filecoin.Addr,
			MaxPrice: config.Filecoin.MaxPrice,
		},
	}
}

func toRPCDealErrors(des []ffs.DealError) []*DealError {
	ret := make([]*DealError, len(des))
	for i, de := range des {
		ret[i] = &DealError{
			ProposalCid: de.ProposalCid.String(),
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
		Cid:     info.Cid.String(),
		Created: info.Created.UnixNano(),
		Hot: &HotInfo{
			Enabled: info.Hot.Enabled,
			Size:    int64(info.Hot.Size),
			Ipfs: &IpfsHotInfo{
				Created: info.Hot.Ipfs.Created.UnixNano(),
			},
		},
		Cold: &ColdInfo{
			Filecoin: &FilInfo{
				DataCid:   info.Cold.Filecoin.DataCid.String(),
				Proposals: make([]*FilStorage, len(info.Cold.Filecoin.Proposals)),
			},
		},
	}
	for i, p := range info.Cold.Filecoin.Proposals {
		cidInfo.Cold.Filecoin.Proposals[i] = &FilStorage{
			ProposalCid:     p.ProposalCid.String(),
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
