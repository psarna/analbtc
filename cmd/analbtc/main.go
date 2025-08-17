package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"analbtc/internal/config"
	"analbtc/pkg/metrics"
)

func main() {
	var (
		configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
		historical = flag.Bool("historical", false, "Run historical data ingestion")
		continuous = flag.Bool("continuous", false, "Run continuous data ingestion")
		stats      = flag.Bool("stats", false, "Show ingestion statistics")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize ingester
	ingester, err := metrics.NewIngester(cfg)
	if err != nil {
		log.Fatalf("Failed to create ingester: %v", err)
	}
	defer ingester.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal, gracefully stopping...")
		cancel()
	}()

	// Execute based on flags
	switch {
	case *stats:
		if err := showStats(ingester); err != nil {
			log.Fatalf("Failed to show stats: %v", err)
		}

	case *historical:
		log.Println("Starting historical data ingestion...")
		if err := ingester.IngestHistoricalData(ctx); err != nil {
			log.Fatalf("Historical ingestion failed: %v", err)
		}
		log.Println("Historical data ingestion completed successfully")

	case *continuous:
		log.Println("Starting continuous data ingestion...")
		if err := ingester.IngestLatestBlocks(ctx); err != nil && err != context.Canceled {
			log.Fatalf("Continuous ingestion failed: %v", err)
		}
		log.Println("Continuous ingestion stopped")

	default:
		// Default: run historical first, then continuous
		log.Println("Running full ingestion (historical + continuous)")
		
		// First run historical ingestion
		if err := ingester.IngestHistoricalData(ctx); err != nil {
			log.Fatalf("Historical ingestion failed: %v", err)
		}
		
		// Then switch to continuous mode
		log.Println("Switching to continuous ingestion mode...")
		if err := ingester.IngestLatestBlocks(ctx); err != nil && err != context.Canceled {
			log.Fatalf("Continuous ingestion failed: %v", err)
		}
		log.Println("Ingestion stopped")
	}
}

func showStats(ingester *metrics.Ingester) error {
	stats, err := ingester.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("=== Analbtc Ingestion Statistics ===")
	fmt.Printf("Blocks in database: %v\n", stats["blocks_in_db"])
	fmt.Printf("Latest DB height: %v\n", stats["latest_db_height"])
	fmt.Printf("Network height: %v\n", stats["network_height"])
	fmt.Printf("Blocks behind: %v\n", stats["blocks_behind"])
	fmt.Printf("Sync percentage: %.2f%%\n", stats["sync_percentage"])

	return nil
}