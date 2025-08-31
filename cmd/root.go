package cmd

import (
	"context"
	"fmt"
	"os"
	"scrapbtc/internal/db"
	"scrapbtc/internal/processor"
	"scrapbtc/internal/rpc"
	"scrapbtc/internal/ui"
	"time"

	"github.com/spf13/cobra"
)

var (
	dbPath     string
	rpcHost    string
	rpcUser    string
	rpcPass    string
	startDate  string
	endDate    string
	workers    int
)

var rootCmd = &cobra.Command{
	Use:   "scrapbtc",
	Short: "Bitcoin blockchain data scraper for investment analysis",
	Long: `A fast, concurrent Bitcoin blockchain scraper that extracts block and transaction data
from Bitcoin Core RPC and stores it in DuckDB for analysis.`,
	RunE: runScraper,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&dbPath, "database", "d", "bitcoin_data.db", "DuckDB database file path")
	rootCmd.Flags().StringVarP(&rpcHost, "host", "H", "localhost:8332", "Bitcoin RPC host and port")
	rootCmd.Flags().StringVarP(&rpcUser, "user", "u", "", "Bitcoin RPC username")
	rootCmd.Flags().StringVarP(&rpcPass, "pass", "p", "", "Bitcoin RPC password")
	rootCmd.Flags().StringVarP(&startDate, "from", "f", "", "Start date (YYYY-MM-DD), default: 1 year ago")
	rootCmd.Flags().StringVarP(&endDate, "to", "t", "", "End date (YYYY-MM-DD), default: today")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 10, "Number of concurrent workers")

	rootCmd.MarkFlagRequired("user")
	rootCmd.MarkFlagRequired("pass")
}

func runScraper(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	database, err := db.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer database.Close()

	rpcClient, err := rpc.NewClient(rpcHost, rpcUser, rpcPass)
	if err != nil {
		return fmt.Errorf("failed to create RPC client: %w", err)
	}
	defer rpcClient.Close()

	startHeight, endHeight, err := calculateHeightRange(rpcClient)
	if err != nil {
		return fmt.Errorf("failed to calculate height range: %w", err)
	}

	fmt.Printf("Processing blocks from height %d to %d (%d blocks total)\n", 
		startHeight, endHeight, endHeight-startHeight+1)

	workerPool := processor.NewWorkerPool(rpcClient, database, workers)
	
	go func() {
		if err := workerPool.ProcessBlockRange(ctx, startHeight, endHeight); err != nil {
			fmt.Fprintf(os.Stderr, "Processing error: %v\n", err)
		}
	}()

	return ui.RunProgressUI(ctx, startHeight, endHeight, workerPool.GetProgressChannel())
}

func calculateHeightRange(rpcClient *rpc.Client) (int64, int64, error) {
	bestHeight, err := rpcClient.GetBestBlockHeight()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get best block height: %w", err)
	}

	endHeight := bestHeight
	if endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end date format: %w", err)
		}
		endHeight = heightFromTimestamp(t)
	}

	startHeight := int64(0)
	if startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start date format: %w", err)
		}
		startHeight = heightFromTimestamp(t)
	} else {
		oneYearAgo := time.Now().AddDate(-1, 0, 0)
		startHeight = heightFromTimestamp(oneYearAgo)
	}

	if startHeight < 0 {
		startHeight = 0
	}
	if endHeight > bestHeight {
		endHeight = bestHeight
	}

	return startHeight, endHeight, nil
}

func heightFromTimestamp(t time.Time) int64 {
	genesisTime := time.Date(2009, 1, 3, 18, 15, 5, 0, time.UTC)
	if t.Before(genesisTime) {
		return 0
	}
	
	blockTime := 10 * time.Minute
	elapsedTime := t.Sub(genesisTime)
	estimatedHeight := int64(elapsedTime / blockTime)
	
	return estimatedHeight
}