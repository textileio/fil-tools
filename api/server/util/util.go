package util

import (
	"fmt"

	userPb "github.com/textileio/powergate/api/gen/powergate/user/v1"
	"github.com/textileio/powergate/ffs"
	"github.com/textileio/powergate/util"
)

// ToRPCStorageInfo converts a StorageInfo to the proto version.
func ToRPCStorageInfo(info ffs.StorageInfo) *userPb.StorageInfo {
	storageInfo := &userPb.StorageInfo{
		JobId:   info.JobID.String(),
		Cid:     util.CidToString(info.Cid),
		Created: info.Created.UnixNano(),
		Hot: &userPb.HotInfo{
			Enabled: info.Hot.Enabled,
			Size:    int64(info.Hot.Size),
			Ipfs: &userPb.IpfsHotInfo{
				Created: info.Hot.Ipfs.Created.UnixNano(),
			},
		},
		Cold: &userPb.ColdInfo{
			Enabled: info.Cold.Enabled,
			Filecoin: &userPb.FilInfo{
				DataCid:   util.CidToString(info.Cold.Filecoin.DataCid),
				Size:      info.Cold.Filecoin.Size,
				Proposals: make([]*userPb.FilStorage, len(info.Cold.Filecoin.Proposals)),
			},
		},
	}
	for i, p := range info.Cold.Filecoin.Proposals {
		var strPieceCid string
		if p.PieceCid.Defined() {
			strPieceCid = util.CidToString(p.PieceCid)
		}
		storageInfo.Cold.Filecoin.Proposals[i] = &userPb.FilStorage{
			DealId:     int64(p.DealID),
			PieceCid:   strPieceCid,
			Renewed:    p.Renewed,
			Duration:   p.Duration,
			StartEpoch: p.StartEpoch,
			Miner:      p.Miner,
			EpochPrice: p.EpochPrice,
		}
	}
	return storageInfo
}

// ToProtoStorageJobs converts a slice of ffs.StorageJobs to proto Jobs.
func ToProtoStorageJobs(jobs []ffs.StorageJob) ([]*userPb.StorageJob, error) {
	var res []*userPb.StorageJob
	for _, job := range jobs {
		j, err := ToRPCJob(job)
		if err != nil {
			return nil, err
		}
		res = append(res, j)
	}
	return res, nil
}

// ToRPCJob converts a job to a proto job.
func ToRPCJob(job ffs.StorageJob) (*userPb.StorageJob, error) {
	var dealInfo []*userPb.DealInfo
	for _, item := range job.DealInfo {
		info := &userPb.DealInfo{
			ActivationEpoch: item.ActivationEpoch,
			DealId:          item.DealID,
			Duration:        item.Duration,
			Message:         item.Message,
			Miner:           item.Miner,
			PieceCid:        item.PieceCID.String(),
			PricePerEpoch:   item.PricePerEpoch,
			ProposalCid:     item.ProposalCid.String(),
			Size:            item.Size,
			StartEpoch:      item.StartEpoch,
			StateId:         item.StateID,
			StateName:       item.StateName,
		}
		dealInfo = append(dealInfo, info)
	}

	var status userPb.JobStatus
	switch job.Status {
	case ffs.Unspecified:
		status = userPb.JobStatus_JOB_STATUS_UNSPECIFIED
	case ffs.Queued:
		status = userPb.JobStatus_JOB_STATUS_QUEUED
	case ffs.Executing:
		status = userPb.JobStatus_JOB_STATUS_EXECUTING
	case ffs.Failed:
		status = userPb.JobStatus_JOB_STATUS_FAILED
	case ffs.Canceled:
		status = userPb.JobStatus_JOB_STATUS_CANCELED
	case ffs.Success:
		status = userPb.JobStatus_JOB_STATUS_SUCCESS
	default:
		return nil, fmt.Errorf("unknown job status: %v", job.Status)
	}
	return &userPb.StorageJob{
		Id:         job.ID.String(),
		ApiId:      job.APIID.String(),
		Cid:        util.CidToString(job.Cid),
		Status:     status,
		ErrorCause: job.ErrCause,
		DealErrors: toRPCDealErrors(job.DealErrors),
		CreatedAt:  job.CreatedAt,
		DealInfo:   dealInfo,
	}, nil
}

func toRPCDealErrors(des []ffs.DealError) []*userPb.DealError {
	ret := make([]*userPb.DealError, len(des))
	for i, de := range des {
		var strProposalCid string
		if de.ProposalCid.Defined() {
			strProposalCid = util.CidToString(de.ProposalCid)
		}
		ret[i] = &userPb.DealError{
			ProposalCid: strProposalCid,
			Miner:       de.Miner,
			Message:     de.Message,
		}
	}
	return ret
}
