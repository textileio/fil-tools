package cmd

import (
	"context"
	"errors"
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
	"github.com/textileio/powergate/ffs"
)

func init() {
	ffsCmd.AddCommand(ffsWatchCmd)
}

var ffsWatchCmd = &cobra.Command{
	Use:   "watch [jobid,...]",
	Short: "Watch for job status updates",
	Long:  `Watch for job status updates`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			Fatal(errors.New("you must provide a comma-separated list of job ids"))
		}

		idStrings := strings.Split(args[0], ",")
		jobIds := make([]ffs.JobID, len(idStrings))
		for i, s := range idStrings {
			jobIds[i] = ffs.JobID(s)
		}

		watchJobIds(jobIds...)
	},
}

func watchJobIds(jobIds ...ffs.JobID) {
	state := make(map[string]*client.WatchJobsEvent, len(jobIds))
	for _, jobID := range jobIds {
		state[jobID.String()] = nil
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
		state[event.Job.ID.String()] = &event
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
			if state[k].Job.Status == ffs.Failed {
				val = fmt.Sprintf("%v %v", displayName(state[k].Job.Status), state[k].Job.ErrCause)
			} else if state[k].Err != nil {
				val = fmt.Sprintf("Error: %v", state[k].Err.Error())
			} else {
				val = displayName(state[k].Job.Status)
			}
			data = append(data, []string{k, val, "", "", ""})
			for _, dealInfo := range state[k].Job.DealInfo {
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
			event.Job.Status == ffs.Executing ||
			event.Job.Status == ffs.Queued {
			processing = true
		}
		if processing && event != nil && event.Err == nil {
			return false
		}
	}
	return true
}

func displayName(s ffs.JobStatus) string {
	name, ok := ffs.JobStatusStr[s]
	if !ok {
		return "Unknown"
	}
	return name
}
