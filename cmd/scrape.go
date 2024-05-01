package cmd

import (
	"encoding/hex"
	"fmt"

	pkg "github.com/shyba/btshow/pkg"
	"github.com/spf13/cobra"
)

var host string

var scrapeCmd = &cobra.Command{
	Use:   "scrape [flags] <infohash1> <infohash2> ...",
	Short: "Scrape one or more infohashes for completed, leechers and peers.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client := pkg.NewTrackerClient(host)
		defer client.Close()

		infohashes := make([]pkg.InfoHash, len(args))
		for idx, arg := range args {
			infohashes[idx] = parseInfohash(arg)
		}

		resp, err := client.Scrape(infohashes...)
		printInfohashResponse(resp)
		if err != nil {
			panic(err)
		}
	},
}

func parseInfohash(infohash string) pkg.InfoHash {
	val, err := hex.DecodeString(infohash)
	if err != nil {
		panic(err)
	}
	return pkg.InfoHash(val[0:20])
}

func printInfohashResponse(response pkg.ScrapeResponse) {
	for infohash := range response {
		fmt.Println(hex.EncodeToString(infohash[:]))
		stat := response[infohash]
		fmt.Printf("Completed: %d\n", stat.Completed)
		fmt.Printf("Leechers: %d\n", stat.Leechers)
		fmt.Printf("Seeders: %d\n", stat.Seeders)
	}
}

func init() {
	scrapeCmd.Flags().StringVarP(&host, "udp-tracker-host", "u", "tracker.opentrackr.org:1337", "UDP tracker host and port (required)")
	scrapeCmd.MarkFlagRequired("host")
	rootCmd.AddCommand(scrapeCmd)
}
