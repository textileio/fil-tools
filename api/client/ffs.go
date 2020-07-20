package client

import (
	"context"
	"fmt"
	"io"
	"time"

	cid "github.com/ipfs/go-cid"
	"github.com/textileio/powergate/deals"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/ffs/api"
	"github.com/textileio/powergate/ffs/rpc"
	"github.com/textileio/powergate/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FFS provides the API to create and interact with an FFS instance.
type FFS struct {
	client rpc.RPCServiceClient
}

// JobEvent represents an event for Watching a job.
type JobEvent struct {
	Job ffs.Job
	Err error
}

// NewAddressOption is a function that changes a NewAddressConfig.
type NewAddressOption func(r *rpc.NewAddrRequest)

// WithMakeDefault specifies if the new address should become the default.
func WithMakeDefault(makeDefault bool) NewAddressOption {
	return func(r *rpc.NewAddrRequest) {
		r.MakeDefault = makeDefault
	}
}

// WithAddressType specifies the type of address to create.
func WithAddressType(addressType string) NewAddressOption {
	return func(r *rpc.NewAddrRequest) {
		r.AddressType = addressType
	}
}

// PushConfigOption mutates a push request.
type PushConfigOption func(r *rpc.PushConfigRequest)

// WithCidConfig overrides the Api default Cid configuration.
func WithCidConfig(c ffs.CidConfig) PushConfigOption {
	return func(r *rpc.PushConfigRequest) {
		r.HasConfig = true
		r.Config = &rpc.CidConfig{
			Cid:  util.CidToString(c.Cid),
			Hot:  toRPCHotConfig(c.Hot),
			Cold: toRPCColdConfig(c.Cold),
		}
	}
}

// WithOverride allows a new push configuration to override an existing one.
// It's used as an extra security measure to avoid unwanted configuration changes.
func WithOverride(override bool) PushConfigOption {
	return func(r *rpc.PushConfigRequest) {
		r.HasOverrideConfig = true
		r.OverrideConfig = override
	}
}

// WatchLogsOption is a function that changes GetLogsConfig.
type WatchLogsOption func(r *rpc.WatchLogsRequest)

// WithJidFilter filters only log messages of a Cid related to
// the Job with id jid.
func WithJidFilter(jid ffs.JobID) WatchLogsOption {
	return func(r *rpc.WatchLogsRequest) {
		r.Jid = jid.String()
	}
}

// WithHistory indicates that prior history logs should
// be sent in the channel before getting real time logs.
func WithHistory(enabled bool) WatchLogsOption {
	return func(r *rpc.WatchLogsRequest) {
		r.History = enabled
	}
}

// LogEvent represents an event for watching cid logs.
type LogEvent struct {
	LogEntry ffs.LogEntry
	Err      error
}

// ListDealRecordsOption updates a ListDealRecordsConfig.
type ListDealRecordsOption func(*rpc.ListDealRecordsConfig)

// WithFromAddrs limits the results deals initiated from the provided wallet addresses.
// If WithDataCids is also provided, this is an AND operation.
func WithFromAddrs(addrs ...string) ListDealRecordsOption {
	return func(c *rpc.ListDealRecordsConfig) {
		c.FromAddrs = addrs
	}
}

// WithDataCids limits the results to deals for the provided data cids.
// If WithFromAddrs is also provided, this is an AND operation.
func WithDataCids(cids ...string) ListDealRecordsOption {
	return func(c *rpc.ListDealRecordsConfig) {
		c.DataCids = cids
	}
}

// WithIncludePending specifies whether or not to include pending deals in the results. Default is false.
// Ignored for ListRetrievalDealRecords.
func WithIncludePending(includePending bool) ListDealRecordsOption {
	return func(c *rpc.ListDealRecordsConfig) {
		c.IncludePending = includePending
	}
}

// WithIncludeFinal specifies whether or not to include final deals in the results. Default is false.
// Ignored for ListRetrievalDealRecords.
func WithIncludeFinal(includeFinal bool) ListDealRecordsOption {
	return func(c *rpc.ListDealRecordsConfig) {
		c.IncludeFinal = includeFinal
	}
}

// WithAscending specifies to sort the results in ascending order. Default is descending order.
// Records are sorted by timestamp.
func WithAscending(ascending bool) ListDealRecordsOption {
	return func(c *rpc.ListDealRecordsConfig) {
		c.Ascending = ascending
	}
}

// Create creates a new FFS instance, returning the instance ID and auth token.
func (f *FFS) Create(ctx context.Context) (string, string, error) {
	r, err := f.client.Create(ctx, &rpc.CreateRequest{})
	if err != nil {
		return "", "", err
	}
	return r.Id, r.Token, nil
}

// ListAPI returns a list of existing API instances.
func (f *FFS) ListAPI(ctx context.Context) ([]ffs.APIID, error) {
	r, err := f.client.ListAPI(ctx, &rpc.ListAPIRequest{})
	if err != nil {
		return nil, err
	}
	res := make([]ffs.APIID, len(r.Instances))
	for i, v := range r.Instances {
		res[i] = ffs.APIID(v)
	}
	return res, nil
}

// ID returns the FFS instance ID.
func (f *FFS) ID(ctx context.Context) (ffs.APIID, error) {
	resp, err := f.client.ID(ctx, &rpc.IDRequest{})
	if err != nil {
		return ffs.EmptyInstanceID, err
	}
	return ffs.APIID(resp.Id), nil
}

// Addrs returns a list of addresses managed by the FFS instance.
func (f *FFS) Addrs(ctx context.Context) ([]api.AddrInfo, error) {
	resp, err := f.client.Addrs(ctx, &rpc.AddrsRequest{})
	if err != nil {
		return nil, err
	}
	addrs := make([]api.AddrInfo, len(resp.Addrs))
	for i, addr := range resp.Addrs {
		addrs[i] = api.AddrInfo{
			Name: addr.Name,
			Addr: addr.Addr,
			Type: addr.Type,
		}
	}
	return addrs, nil
}

// DefaultConfig returns the default storage config.
func (f *FFS) DefaultConfig(ctx context.Context) (ffs.DefaultConfig, error) {
	resp, err := f.client.DefaultConfig(ctx, &rpc.DefaultConfigRequest{})
	if err != nil {
		return ffs.DefaultConfig{}, err
	}
	return ffs.DefaultConfig{
		Hot: ffs.HotConfig{
			Enabled:       resp.DefaultConfig.Hot.Enabled,
			AllowUnfreeze: resp.DefaultConfig.Hot.AllowUnfreeze,
			Ipfs: ffs.IpfsConfig{
				AddTimeout: int(resp.DefaultConfig.Hot.Ipfs.AddTimeout),
			},
		},
		Cold: ffs.ColdConfig{
			Enabled: resp.DefaultConfig.Cold.Enabled,
			Filecoin: ffs.FilConfig{
				RepFactor:       int(resp.DefaultConfig.Cold.Filecoin.RepFactor),
				DealMinDuration: resp.DefaultConfig.Cold.Filecoin.DealMinDuration,
				ExcludedMiners:  resp.DefaultConfig.Cold.Filecoin.ExcludedMiners,
				CountryCodes:    resp.DefaultConfig.Cold.Filecoin.CountryCodes,
				TrustedMiners:   resp.DefaultConfig.Cold.Filecoin.TrustedMiners,
				Renew: ffs.FilRenew{
					Enabled:   resp.DefaultConfig.Cold.Filecoin.Renew.Enabled,
					Threshold: int(resp.DefaultConfig.Cold.Filecoin.Renew.Threshold),
				},
				Addr:     resp.DefaultConfig.Cold.Filecoin.Addr,
				MaxPrice: resp.DefaultConfig.Cold.Filecoin.MaxPrice,
			},
		},
		Repairable: resp.DefaultConfig.Repairable,
	}, nil
}

// NewAddr created a new wallet address managed by the FFS instance.
func (f *FFS) NewAddr(ctx context.Context, name string, options ...NewAddressOption) (string, error) {
	r := &rpc.NewAddrRequest{Name: name}
	for _, opt := range options {
		opt(r)
	}
	resp, err := f.client.NewAddr(ctx, r)
	return resp.Addr, err
}

// GetDefaultCidConfig returns a CidConfig built from the default storage config and prepped for the provided cid.
func (f *FFS) GetDefaultCidConfig(ctx context.Context, c cid.Cid) (ffs.CidConfig, error) {
	res, err := f.client.GetDefaultCidConfig(ctx, &rpc.GetDefaultCidConfigRequest{Cid: util.CidToString(c)})
	if err != nil {
		return ffs.CidConfig{}, err
	}
	resCid, err := util.CidFromString(res.Config.Cid)
	if err != nil {
		return ffs.CidConfig{}, err
	}
	return ffs.CidConfig{
		Cid:        resCid,
		Repairable: res.Config.Repairable,
		Hot: ffs.HotConfig{
			AllowUnfreeze: res.Config.Hot.AllowUnfreeze,
			Enabled:       res.Config.Hot.Enabled,
			Ipfs: ffs.IpfsConfig{
				AddTimeout: int(res.Config.Hot.Ipfs.AddTimeout),
			},
		},
		Cold: ffs.ColdConfig{
			Enabled: res.Config.Cold.Enabled,
			Filecoin: ffs.FilConfig{
				RepFactor:       int(res.Config.Cold.Filecoin.RepFactor),
				Addr:            res.Config.Cold.Filecoin.Addr,
				CountryCodes:    res.Config.Cold.Filecoin.CountryCodes,
				DealMinDuration: res.Config.Cold.Filecoin.DealMinDuration,
				ExcludedMiners:  res.Config.Cold.Filecoin.ExcludedMiners,
				Renew: ffs.FilRenew{
					Enabled:   res.Config.Cold.Filecoin.Renew.Enabled,
					Threshold: int(res.Config.Cold.Filecoin.Renew.Threshold),
				},
				TrustedMiners: res.Config.Cold.Filecoin.TrustedMiners,
				MaxPrice:      res.Config.Cold.Filecoin.MaxPrice,
			},
		},
	}, nil
}

// GetCidConfig gets the current config for a cid.
func (f *FFS) GetCidConfig(ctx context.Context, c cid.Cid) (*rpc.GetCidConfigResponse, error) {
	return f.client.GetCidConfig(ctx, &rpc.GetCidConfigRequest{Cid: util.CidToString(c)})
}

// SetDefaultConfig sets the default storage config.
func (f *FFS) SetDefaultConfig(ctx context.Context, config ffs.DefaultConfig) error {
	req := &rpc.SetDefaultConfigRequest{
		Config: &rpc.DefaultConfig{
			Hot:        toRPCHotConfig(config.Hot),
			Cold:       toRPCColdConfig(config.Cold),
			Repairable: config.Repairable,
		},
	}
	_, err := f.client.SetDefaultConfig(ctx, req)
	return err
}

// Show returns information about the current storage state of a cid.
func (f *FFS) Show(ctx context.Context, c cid.Cid) (*rpc.ShowResponse, error) {
	return f.client.Show(ctx, &rpc.ShowRequest{
		Cid: util.CidToString(c),
	})
}

// Info returns information about the FFS instance.
func (f *FFS) Info(ctx context.Context) (api.InstanceInfo, error) {
	res, err := f.client.Info(ctx, &rpc.InfoRequest{})
	if err != nil {
		return api.InstanceInfo{}, err
	}

	balances := make([]api.BalanceInfo, len(res.Info.Balances))
	for i, bal := range res.Info.Balances {
		balances[i] = api.BalanceInfo{
			AddrInfo: api.AddrInfo{
				Name: bal.Addr.Name,
				Addr: bal.Addr.Addr,
				Type: bal.Addr.Type,
			},
			Balance: uint64(bal.Balance),
		}
	}

	pins := make([]cid.Cid, len(res.Info.Pins))
	for i, pin := range res.Info.Pins {
		c, err := util.CidFromString(pin)
		if err != nil {
			return api.InstanceInfo{}, err
		}
		pins[i] = c
	}

	return api.InstanceInfo{
		ID: ffs.APIID(res.Info.Id),
		DefaultConfig: ffs.DefaultConfig{
			Hot: ffs.HotConfig{
				Enabled:       res.Info.DefaultConfig.Hot.Enabled,
				AllowUnfreeze: res.Info.DefaultConfig.Hot.AllowUnfreeze,
				Ipfs: ffs.IpfsConfig{
					AddTimeout: int(res.Info.DefaultConfig.Hot.Ipfs.AddTimeout),
				},
			},
			Cold: ffs.ColdConfig{
				Enabled: res.Info.DefaultConfig.Cold.Enabled,
				Filecoin: ffs.FilConfig{
					RepFactor:       int(res.Info.DefaultConfig.Cold.Filecoin.RepFactor),
					DealMinDuration: res.Info.DefaultConfig.Cold.Filecoin.DealMinDuration,
					ExcludedMiners:  res.Info.DefaultConfig.Cold.Filecoin.ExcludedMiners,
					TrustedMiners:   res.Info.DefaultConfig.Cold.Filecoin.TrustedMiners,
					CountryCodes:    res.Info.DefaultConfig.Cold.Filecoin.CountryCodes,
					Renew: ffs.FilRenew{
						Enabled:   res.Info.DefaultConfig.Cold.Filecoin.Renew.Enabled,
						Threshold: int(res.Info.DefaultConfig.Cold.Filecoin.Renew.Threshold),
					},
					Addr:     res.Info.DefaultConfig.Cold.Filecoin.Addr,
					MaxPrice: res.Info.DefaultConfig.Cold.Filecoin.MaxPrice,
				},
			},
			Repairable: res.Info.DefaultConfig.Repairable,
		},
		Balances: balances,
		Pins:     pins,
	}, nil
}

// CancelJob signals that the executing Job with JobID jid should be
// canceled.
func (f *FFS) CancelJob(ctx context.Context, jid ffs.JobID) error {
	_, err := f.client.CancelJob(ctx, &rpc.CancelJobRequest{Jid: jid.String()})
	return err
}

// WatchJobs pushes JobEvents to the provided channel. The provided channel will be owned
// by the client after the call, so it shouldn't be closed by the client. To stop receiving
// events, the provided ctx should be canceled. If an error occurs, it will be returned
// in the Err field of JobEvent and the channel will be closed.
func (f *FFS) WatchJobs(ctx context.Context, ch chan<- JobEvent, jids ...ffs.JobID) error {
	jidStrings := make([]string, len(jids))
	for i, jid := range jids {
		jidStrings[i] = jid.String()
	}

	stream, err := f.client.WatchJobs(ctx, &rpc.WatchJobsRequest{Jids: jidStrings})
	if err != nil {
		return err
	}
	go func() {
		for {
			reply, err := stream.Recv()
			if err == io.EOF || status.Code(err) == codes.Canceled {
				close(ch)
				break
			}
			if err != nil {
				ch <- JobEvent{Err: err}
				close(ch)
				break
			}

			c, err := util.CidFromString(reply.Job.Cid)
			if err != nil {
				ch <- JobEvent{Err: err}
				close(ch)
				break
			}
			dealErrors, err := fromRPCDealErrors(reply.Job.DealErrors)
			if err != nil {
				ch <- JobEvent{Err: err}
				close(ch)
				break
			}
			var status ffs.JobStatus
			switch reply.Job.Status {
			case rpc.JobStatus_JOB_STATUS_QUEUED:
				status = ffs.Queued
			case rpc.JobStatus_JOB_STATUS_EXECUTING:
				status = ffs.Executing
			case rpc.JobStatus_JOB_STATUS_FAILED:
				status = ffs.Failed
			case rpc.JobStatus_JOB_STATUS_CANCELED:
				status = ffs.Canceled
			case rpc.JobStatus_JOB_STATUS_SUCCESS:
				status = ffs.Success
			default:
				status = ffs.Unspecified
			}
			job := ffs.Job{
				ID:         ffs.JobID(reply.Job.Id),
				APIID:      ffs.APIID(reply.Job.ApiId),
				Cid:        c,
				Status:     status,
				ErrCause:   reply.Job.ErrCause,
				DealErrors: dealErrors,
			}
			ch <- JobEvent{Job: job}
		}
	}()
	return nil
}

// Replace pushes a CidConfig of c2 equal to c1, and removes c1. This operation
// is more efficient than manually removing and adding in two separate operations.
func (f *FFS) Replace(ctx context.Context, c1 cid.Cid, c2 cid.Cid) (ffs.JobID, error) {
	resp, err := f.client.Replace(ctx, &rpc.ReplaceRequest{Cid1: util.CidToString(c1), Cid2: util.CidToString(c2)})
	if err != nil {
		return ffs.EmptyJobID, err
	}
	return ffs.JobID(resp.JobId), nil
}

// PushConfig push a new configuration for the Cid in the Hot and Cold layers.
func (f *FFS) PushConfig(ctx context.Context, c cid.Cid, opts ...PushConfigOption) (ffs.JobID, error) {
	req := &rpc.PushConfigRequest{Cid: util.CidToString(c)}
	for _, opt := range opts {
		opt(req)
	}

	resp, err := f.client.PushConfig(ctx, req)
	if err != nil {
		return ffs.EmptyJobID, err
	}

	return ffs.JobID(resp.JobId), nil
}

// Remove removes a Cid from being tracked as an active storage. The Cid should have
// both Hot and Cold storage disabled, if that isn't the case it will return ErrActiveInStorage.
func (f *FFS) Remove(ctx context.Context, c cid.Cid) error {
	_, err := f.client.Remove(ctx, &rpc.RemoveRequest{Cid: util.CidToString(c)})
	return err
}

// Get returns an io.Reader for reading a stored Cid from the Hot Storage.
func (f *FFS) Get(ctx context.Context, c cid.Cid) (io.Reader, error) {
	stream, err := f.client.Get(ctx, &rpc.GetRequest{
		Cid: util.CidToString(c),
	})
	if err != nil {
		return nil, err
	}
	reader, writer := io.Pipe()
	go func() {
		for {
			reply, err := stream.Recv()
			if err == io.EOF {
				_ = writer.Close()
				break
			} else if err != nil {
				_ = writer.CloseWithError(err)
				break
			}
			_, err = writer.Write(reply.GetChunk())
			if err != nil {
				_ = writer.CloseWithError(err)
				break
			}
		}
	}()

	return reader, nil
}

// WatchLogs pushes human-friendly messages about Cid executions. The method is blocking
// and will continue to send messages until the context is canceled. The provided channel
// is owned by the method and must not be closed.
func (f *FFS) WatchLogs(ctx context.Context, ch chan<- LogEvent, c cid.Cid, opts ...WatchLogsOption) error {
	r := &rpc.WatchLogsRequest{Cid: util.CidToString(c)}
	for _, opt := range opts {
		opt(r)
	}

	stream, err := f.client.WatchLogs(ctx, r)
	if err != nil {
		return err
	}
	go func() {
		for {
			reply, err := stream.Recv()
			if err == io.EOF || status.Code(err) == codes.Canceled {
				close(ch)
				break
			}
			if err != nil {
				ch <- LogEvent{Err: err}
				close(ch)
				break
			}

			cid, err := util.CidFromString(reply.LogEntry.Cid)
			if err != nil {
				ch <- LogEvent{Err: err}
				close(ch)
				break
			}

			entry := ffs.LogEntry{
				Cid:       cid,
				Timestamp: time.Unix(reply.LogEntry.Time, 0),
				Jid:       ffs.JobID(reply.LogEntry.Jid),
				Msg:       reply.LogEntry.Msg,
			}
			ch <- LogEvent{LogEntry: entry}
		}
	}()
	return nil
}

// SendFil sends fil from a managed address to any another address, returns immediately but funds are sent asynchronously.
func (f *FFS) SendFil(ctx context.Context, from string, to string, amount int64) error {
	req := &rpc.SendFilRequest{
		From:   from,
		To:     to,
		Amount: amount,
	}
	_, err := f.client.SendFil(ctx, req)
	return err
}

// Close terminates the running FFS instance.
func (f *FFS) Close(ctx context.Context) error {
	_, err := f.client.Close(ctx, &rpc.CloseRequest{})
	return err
}

// AddToHot allows you to add data to the Hot layer in preparation for pushing a cid config.
func (f *FFS) AddToHot(ctx context.Context, data io.Reader) (*cid.Cid, error) {
	stream, err := f.client.AddToHot(ctx)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, 1024*32) // 32KB
	for {
		bytesRead, err := data.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		sendErr := stream.Send(&rpc.AddToHotRequest{Chunk: buffer[:bytesRead]})
		if sendErr != nil {
			if sendErr == io.EOF {
				var noOp interface{}
				return nil, stream.RecvMsg(noOp)
			}
			return nil, sendErr
		}
		if err == io.EOF {
			break
		}
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}

	cid, err := util.CidFromString(reply.GetCid())
	if err != nil {
		return nil, err
	}
	return &cid, nil
}

// ListPayChannels returns a list of payment channels.
func (f *FFS) ListPayChannels(ctx context.Context) ([]ffs.PaychInfo, error) {
	resp, err := f.client.ListPayChannels(ctx, &rpc.ListPayChannelsRequest{})
	if err != nil {
		return []ffs.PaychInfo{}, err
	}
	infos := make([]ffs.PaychInfo, len(resp.PayChannels))
	for i, info := range resp.PayChannels {
		infos[i] = fromRPCPaychInfo(info)
	}
	return infos, nil
}

// CreatePayChannel creates a new payment channel.
func (f *FFS) CreatePayChannel(ctx context.Context, from string, to string, amount uint64) (ffs.PaychInfo, cid.Cid, error) {
	req := &rpc.CreatePayChannelRequest{
		From:   from,
		To:     to,
		Amount: amount,
	}
	resp, err := f.client.CreatePayChannel(ctx, req)
	if err != nil {
		return ffs.PaychInfo{}, cid.Undef, err
	}
	messageCid, err := util.CidFromString(resp.ChannelMessageCid)
	if err != nil {
		return ffs.PaychInfo{}, cid.Undef, err
	}
	return fromRPCPaychInfo(resp.PayChannel), messageCid, nil
}

// RedeemPayChannel redeems a payment channel.
func (f *FFS) RedeemPayChannel(ctx context.Context, addr string) error {
	req := &rpc.RedeemPayChannelRequest{PayChannelAddr: addr}
	_, err := f.client.RedeemPayChannel(ctx, req)
	return err
}

// ListStorageDealRecords returns a list of storage deals for the FFS instance according to the provided options.
func (f *FFS) ListStorageDealRecords(ctx context.Context, opts ...ListDealRecordsOption) ([]deals.StorageDealRecord, error) {
	conf := &rpc.ListDealRecordsConfig{}
	for _, opt := range opts {
		opt(conf)
	}
	res, err := f.client.ListStorageDealRecords(ctx, &rpc.ListStorageDealRecordsRequest{Config: conf})
	if err != nil {
		return nil, fmt.Errorf("calling ListStorageDealRecords: %v", err)
	}
	ret, err := fromRPCStorageDealRecords(res.Records)
	if err != nil {
		return nil, fmt.Errorf("processing response deal records: %v", err)
	}
	return ret, nil
}

// ListRetrievalDealRecords returns a list of retrieval deals for the FFS instance according to the provided options.
func (f *FFS) ListRetrievalDealRecords(ctx context.Context, opts ...ListDealRecordsOption) ([]deals.RetrievalDealRecord, error) {
	conf := &rpc.ListDealRecordsConfig{}
	for _, opt := range opts {
		opt(conf)
	}
	res, err := f.client.ListRetrievalDealRecords(ctx, &rpc.ListRetrievalDealRecordsRequest{Config: conf})
	if err != nil {
		return nil, fmt.Errorf("calling ListRetrievalDealRecords: %v", err)
	}
	ret, err := fromRPCRetrievalDealRecords(res.Records)
	if err != nil {
		return nil, fmt.Errorf("processing response deal records: %v", err)
	}
	return ret, nil
}

func toRPCHotConfig(config ffs.HotConfig) *rpc.HotConfig {
	return &rpc.HotConfig{
		Enabled:       config.Enabled,
		AllowUnfreeze: config.AllowUnfreeze,
		Ipfs: &rpc.IpfsConfig{
			AddTimeout: int64(config.Ipfs.AddTimeout),
		},
	}
}

func toRPCColdConfig(config ffs.ColdConfig) *rpc.ColdConfig {
	return &rpc.ColdConfig{
		Enabled: config.Enabled,
		Filecoin: &rpc.FilConfig{
			RepFactor:       int64(config.Filecoin.RepFactor),
			DealMinDuration: config.Filecoin.DealMinDuration,
			ExcludedMiners:  config.Filecoin.ExcludedMiners,
			TrustedMiners:   config.Filecoin.TrustedMiners,
			CountryCodes:    config.Filecoin.CountryCodes,
			Renew: &rpc.FilRenew{
				Enabled:   config.Filecoin.Renew.Enabled,
				Threshold: int64(config.Filecoin.Renew.Threshold),
			},
			Addr: config.Filecoin.Addr,
		},
	}
}

func fromRPCDealErrors(des []*rpc.DealError) ([]ffs.DealError, error) {
	res := make([]ffs.DealError, len(des))
	for i, de := range des {
		var propCid cid.Cid
		if de.ProposalCid != "" && de.ProposalCid != "b" {
			var err error
			propCid, err = util.CidFromString(de.ProposalCid)
			if err != nil {
				return nil, fmt.Errorf("proposal cid is invalid")
			}
		}
		res[i] = ffs.DealError{
			ProposalCid: propCid,
			Miner:       de.Miner,
			Message:     de.Message,
		}
	}
	return res, nil
}

func fromRPCPaychInfo(info *rpc.PaychInfo) ffs.PaychInfo {
	var direction ffs.PaychDir
	switch info.Direction {
	case rpc.Direction_DIRECTION_INBOUND:
		direction = ffs.PaychDirInbound
	case rpc.Direction_DIRECTION_OUTBOUND:
		direction = ffs.PaychDirOutbound
	default:
		direction = ffs.PaychDirUnspecified
	}
	return ffs.PaychInfo{
		CtlAddr:   info.CtlAddr,
		Addr:      info.Addr,
		Direction: direction,
	}
}

func fromRPCStorageDealRecords(records []*rpc.StorageDealRecord) ([]deals.StorageDealRecord, error) {
	var ret []deals.StorageDealRecord
	for _, rpcRecord := range records {
		if rpcRecord.DealInfo == nil {
			continue
		}
		rootCid, err := util.CidFromString(rpcRecord.RootCid)
		if err != nil {
			return nil, err
		}
		record := deals.StorageDealRecord{
			RootCid: rootCid,
			Addr:    rpcRecord.Addr,
			Time:    rpcRecord.Time,
			Pending: rpcRecord.Pending,
		}
		proposalCid, err := util.CidFromString(rpcRecord.DealInfo.ProposalCid)
		if err != nil {
			return nil, err
		}
		pieceCid, err := util.CidFromString(rpcRecord.DealInfo.PieceCid)
		if err != nil {
			return nil, err
		}
		record.DealInfo = deals.StorageDealInfo{
			ProposalCid:     proposalCid,
			StateID:         rpcRecord.DealInfo.StateId,
			StateName:       rpcRecord.DealInfo.StateName,
			Miner:           rpcRecord.DealInfo.Miner,
			PieceCID:        pieceCid,
			Size:            rpcRecord.DealInfo.Size,
			PricePerEpoch:   rpcRecord.DealInfo.PricePerEpoch,
			StartEpoch:      rpcRecord.DealInfo.StartEpoch,
			Duration:        rpcRecord.DealInfo.Duration,
			DealID:          rpcRecord.DealInfo.DealId,
			ActivationEpoch: rpcRecord.DealInfo.ActivationEpoch,
			Message:         rpcRecord.DealInfo.Msg,
		}
		ret = append(ret, record)
	}
	return ret, nil
}

func fromRPCRetrievalDealRecords(records []*rpc.RetrievalDealRecord) ([]deals.RetrievalDealRecord, error) {
	var ret []deals.RetrievalDealRecord
	for _, rpcRecord := range records {
		if rpcRecord.DealInfo == nil {
			continue
		}
		record := deals.RetrievalDealRecord{
			Addr: rpcRecord.Addr,
			Time: rpcRecord.Time,
		}
		rootCid, err := util.CidFromString(rpcRecord.DealInfo.RootCid)
		if err != nil {
			return nil, err
		}
		record.DealInfo = deals.RetrievalDealInfo{
			RootCid:                 rootCid,
			Size:                    rpcRecord.DealInfo.Size,
			MinPrice:                rpcRecord.DealInfo.MinPrice,
			PaymentInterval:         rpcRecord.DealInfo.PaymentInterval,
			PaymentIntervalIncrease: rpcRecord.DealInfo.PaymentIntervalIncrease,
			Miner:                   rpcRecord.DealInfo.Miner,
			MinerPeerID:             rpcRecord.DealInfo.MinerPeerId,
		}
		ret = append(ret, record)
	}
	return ret, nil
}
