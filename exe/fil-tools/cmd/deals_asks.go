package cmd

import (
	"context"
	"errors"
	"os"
	"strconv"

	"github.com/caarlos0/spin"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/filecoin/index/ask"
)

func init() {
	asksCmd.Flags().Uint64P("maxPrice", "m", 0, "max price of the asks to query")
	asksCmd.Flags().IntP("pieceSize", "p", 0, "piece size of the asks to query")
	asksCmd.Flags().IntP("limit", "l", -1, "limit the number of results")
	asksCmd.Flags().IntP("offset", "o", -1, "offset of results")

	dealsCmd.AddCommand(asksCmd)
}

var asksCmd = &cobra.Command{
	Use:   "asks",
	Short: "Print the available asks",
	Long:  `Print the available asks`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		checkErr(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
		defer cancel()

		mp := viper.GetUint64("maxPrice")
		ps := viper.GetUint64("pieceSize")
		l := viper.GetInt("limit")
		o := viper.GetInt("offset")

		if mp == 0 {
			Fatal(errors.New("maxPrice must be > 0"))
		}

		if ps == 0 {
			Fatal(errors.New("pieceSize must be > 0"))
		}

		q := ask.Query{
			MaxPrice:  mp,
			PieceSize: ps,
			Limit:     l,
			Offset:    o,
		}

		s := spin.New("%s Querying network for available storage asks...")
		s.Start()
		asks, err := fcClient.Deals.AvailableAsks(ctx, q)
		s.Stop()
		checkErr(err)

		if len(asks) > 0 {
			data := make([][]string, len(asks))
			for i, a := range asks {
				data[i] = []string{
					a.Miner,
					strconv.Itoa(int(a.Price)),
					strconv.Itoa(int(a.MinPieceSize)),
					strconv.Itoa(int(a.Timestamp)),
					strconv.Itoa(int(a.Expiry)),
				}
			}
			RenderTable(os.Stdout, []string{"miner", "price", "min piece size", "timestamp", "expiry"}, data)
		}

		Message("Found %d asks", aurora.White(len(asks)).Bold())

	},
}
