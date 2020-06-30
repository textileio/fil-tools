package cmd

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/spin"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	dealsCmd.AddCommand(retrievalCmd)
}

var retrievalCmd = &cobra.Command{
	Use:   "retrieval",
	Short: "List retrieval records",
	Long:  `List retrieval records`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		s := spin.New("%s Getting retrieval records...")
		s.Start()
		res, err := fcClient.Deals.RetrievalRecords(ctx)
		s.Stop()
		checkErr(err)

		if len(res) > 0 {
			data := make([][]string, len(res))
			for i, r := range res {
				t := time.Unix(r.Time, 0)
				data[i] = []string{
					t.Format("01/02/06 15:04 MST"),
					r.Addr,
					r.RetrievalInfo.Miner,
					r.RetrievalInfo.PieceCID.String(),
					strconv.Itoa(int(r.RetrievalInfo.Size)),
				}
			}
			RenderTable(os.Stdout, []string{"time", "addr", "miner", "piece cid", "size"}, data)
		}
		Message("Found %d retrievals", aurora.White(len(res)).Bold())
	},
}
