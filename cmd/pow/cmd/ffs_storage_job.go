package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	ffsCmd.AddCommand(ffsStorageJobsCmd)
	ffsStorageJobsCmd.AddCommand(
		ffsGetStorageJobCmd,
		ffsQueuedStorageJobsCmd,
		ffsExecutingStorageJobsCmd,
		ffsLatestFinalStorageJobsCmd,
		ffsLatestSuccessfulStorageJobsCmd,
		ffsStorageJobsSummaryCmd,
		ffsStorageConfigForJobCmd,
	)
}

var ffsStorageJobsCmd = &cobra.Command{
	Use:     "storage-jobs",
	Aliases: []string{"storage-job"},
	Short:   "Provides commands to query for storage jobs in various states",
	Long:    `Provides commands to query for storage jobs in various statess`,
}

var ffsGetStorageJobCmd = &cobra.Command{
	Use:     "get [jobid]",
	Aliases: []string{"storage-job"},
	Short:   "Get a storage job's current status",
	Long:    `Get a storage job's current status`,
	Args:    cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.StorageJob(mustAuthCtx(ctx), args[0])
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res.Job)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsQueuedStorageJobsCmd = &cobra.Command{
	Use:   "queued [optional cid1,cid2,...]",
	Short: "List queued storage jobs",
	Long:  `List queued storage jobs`,
	Args:  cobra.RangeArgs(0, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cids []string
		if len(args) > 0 {
			cids = strings.Split(args[0], ",")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.QueuedStorageJobs(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsExecutingStorageJobsCmd = &cobra.Command{
	Use:   "executing [optional cid1,cid2,...]",
	Short: "List executing storage jobs",
	Long:  `List executing storage jobs`,
	Args:  cobra.RangeArgs(0, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cids []string
		if len(args) > 0 {
			cids = strings.Split(args[0], ",")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.ExecutingStorageJobs(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsLatestFinalStorageJobsCmd = &cobra.Command{
	Use:   "latest-final [optional cid1,cid2,...]",
	Short: "List the latest final storage jobs",
	Long:  `List the latest final storage jobs`,
	Args:  cobra.RangeArgs(0, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cids []string
		if len(args) > 0 {
			cids = strings.Split(args[0], ",")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.LatestFinalStorageJobs(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsLatestSuccessfulStorageJobsCmd = &cobra.Command{
	Use:   "latest-successful [optional cid1,cid2,...]",
	Short: "List the latest successful storage jobs",
	Long:  `List the latest successful storage jobs`,
	Args:  cobra.RangeArgs(0, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cids []string
		if len(args) > 0 {
			cids = strings.Split(args[0], ",")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.LatestSuccessfulStorageJobs(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsStorageJobsSummaryCmd = &cobra.Command{
	Use:   "summary [optional cid1,cid2,...]",
	Short: "Give a summary of storage jobs in all states",
	Long:  `Give a summary of storage jobs in all states`,
	Args:  cobra.RangeArgs(0, 1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		var cids []string
		if len(args) > 0 {
			cids = strings.Split(args[0], ",")
		}

		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.FFS.StorageJobsSummary(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}

var ffsStorageConfigForJobCmd = &cobra.Command{
	Use:   "storage-config [job-id]",
	Short: "Get the StorageConfig associated with the specified job",
	Long:  `Get the StorageConfig associated with the specified job`,
	Args:  cobra.ExactArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		res, err := fcClient.Jobs.StorageConfigForJob(mustAuthCtx(ctx), args[0])
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res.StorageConfig)
		checkErr(err)

		fmt.Println(string(json))
	},
}
