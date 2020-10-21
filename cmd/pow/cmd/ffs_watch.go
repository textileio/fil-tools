package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/apoorvam/goterminal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/powergate/api/client"
	"github.com/textileio/powergate/ffs/rpc"
)

func init() {
	ffsCmd.AddCommand(ffsWatchCmd)
}

var ffsWatchCmd = &cobra.Command{
	Use:   "watch [jobid,...]",
	Short: "Watch for job status updates",
	Long:  `Watch for job status updates`,
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		jobIds := strings.Split(args[0], ",")
		watchJobIds(jobIds...)
	},
}

func watchJobIds(jobIds ...string) {
	state := make(map[string]*client.WatchJobsEvent, len(jobIds))
	for _, jobID := range jobIds {
		state[jobID] = nil
	}

	writer := goterminal.New(os.Stdout)

	updateJobsOutput(writer, state)

	ch := make(chan client.WatchJobsEvent)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := fcClient.FFS.WatchJobs(mustAuthCtx(ctx), ch, jobIds...)
	checkErr(err)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		os.Exit(0)
	}()

	for {
		event, ok := <-ch
		if !ok {
			break
		}
		state[event.Res.Job.Id] = &event
		updateJobsOutput(writer, state)
		if jobsComplete(state) {
			break
		}
	}
}

func updateJobsOutput(writer *goterminal.Writer, state map[string]*client.WatchJobsEvent) {
	keys := make([]string, 0, len(state))
	for k := range state {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var data [][]string
	for _, k := range keys {
		if state[k] != nil {
			var val string
			if state[k].Res.Job.Status == rpc.JobStatus_JOB_STATUS_FAILED {
				val = fmt.Sprintf("%v %v", displayName(state[k].Res.Job.Status), state[k].Res.Job.ErrCause)
			} else if state[k].Err != nil {
				val = fmt.Sprintf("Error: %v", state[k].Err.Error())
			} else {
				val = displayName(state[k].Res.Job.Status)
			}
			data = append(data, []string{k, val, "", "", ""})
			for _, dealInfo := range state[k].Res.Job.DealInfo {
				data = append(data, []string{"", "", dealInfo.Miner, strconv.FormatUint(dealInfo.PricePerEpoch, 10), dealInfo.StateName})
			}
		} else {
			data = append(data, []string{k, "awaiting state", "", "", ""})
		}
	}

	RenderTable(writer, []string{"Job id", "Status", "Miner", "Price", "Deal Status"}, data)

	writer.Clear()
	_ = writer.Print()
}

func jobsComplete(state map[string]*client.WatchJobsEvent) bool {
	for _, event := range state {
		processing := false
		if event == nil ||
			event.Res.Job.Status == rpc.JobStatus_JOB_STATUS_EXECUTING ||
			event.Res.Job.Status == rpc.JobStatus_JOB_STATUS_QUEUED {
			processing = true
		}
		if processing && event != nil && event.Err == nil {
			return false
		}
	}
	return true
}

func displayName(s rpc.JobStatus) string {
	name, ok := rpc.JobStatus_name[int32(s)]
	if !ok {
		return "Unknown"
	}
	return name
}
