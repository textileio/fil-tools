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
	storageJobsCmd.AddCommand(storageJobsExecutingCmd)
}

var storageJobsExecutingCmd = &cobra.Command{
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

		res, err := powClient.StorageJobs.Executing(mustAuthCtx(ctx), cids...)
		checkErr(err)

		json, err := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}.Marshal(res)
		checkErr(err)

		fmt.Println(string(json))
	},
}
