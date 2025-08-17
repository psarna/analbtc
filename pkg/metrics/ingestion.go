package metrics

import (
	"context"
	"fmt"
	"log"
	"time"

	"analbtc/internal/config"
	"analbtc/internal/db"
	"analbtc/internal/rpc"
	"analbtc/pkg/models"
)

type Ingester struct {
	rpcClient *rpc.Client
	database  *db.DuckDB
	config    *config.Config
}

func NewIngester(cfg *config.Config) (*Ingester, error) {
	rpcClient := rpc.NewClient(cfg.Bitcoin.RPC.URL())
	
	database, err := db.NewDuckDB(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &Ingester{
		rpcClient: rpcClient,
		database:  database,
		config:    cfg,
	}, nil
}

func (i *Ingester) Close() error {
	return i.database.Close()
}

func (i *Ingester) IngestHistoricalData(ctx context.Context) error {
	startDate, err := time.Parse("2006-01-02", i.config.Ingestion.StartDate)
	if err != nil {
		return fmt.Errorf("failed to parse start date: %w", err)
	}

	log.Printf("Starting historical data ingestion from %s", startDate.Format("2006-01-02"))

	// Get the block height closest to the start date
	startHeight, err := i.rpcClient.GetBlockHeightByTime(startDate)
	if err != nil {
		return fmt.Errorf("failed to get start block height: %w", err)
	}

	// Get current latest block height from Bitcoin network
	currentHeight, err := i.rpcClient.GetBlockCount()
	if err != nil {
		return fmt.Errorf("failed to get current block count: %w", err)
	}

	// Get latest block height in our database
	dbHeight, err := i.database.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get latest database block height: %w", err)
	}

	// Determine where to start ingestion
	if dbHeight > startHeight {
		startHeight = dbHeight + 1
		log.Printf("Database already contains blocks up to height %d, starting from %d", dbHeight, startHeight)
	}

	log.Printf("Ingesting blocks from height %d to %d (%d blocks)", startHeight, currentHeight, currentHeight-startHeight+1)

	// Ingest blocks in batches
	for height := startHeight; height <= currentHeight; {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batchEnd := height + int64(i.config.Ingestion.BatchSize) - 1
		if batchEnd > currentHeight {
			batchEnd = currentHeight
		}

		if err := i.ingestBlockRange(height, batchEnd); err != nil {
			return fmt.Errorf("failed to ingest blocks %d-%d: %w", height, batchEnd, err)
		}

		log.Printf("Ingested blocks %d-%d", height, batchEnd)
		height = batchEnd + 1
	}

	log.Printf("Historical data ingestion completed. Ingested %d blocks", currentHeight-startHeight+1)
	return nil
}

func (i *Ingester) IngestLatestBlocks(ctx context.Context) error {
	log.Printf("Starting continuous ingestion with %s polling interval", i.config.Ingestion.PollInterval)

	ticker := time.NewTicker(i.config.Ingestion.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := i.checkAndIngestNewBlocks(); err != nil {
				log.Printf("Error checking for new blocks: %v", err)
				continue
			}
		}
	}
}

func (i *Ingester) checkAndIngestNewBlocks() error {
	// Get current latest block height from Bitcoin network
	currentHeight, err := i.rpcClient.GetBlockCount()
	if err != nil {
		return fmt.Errorf("failed to get current block count: %w", err)
	}

	// Get latest block height in our database
	dbHeight, err := i.database.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get latest database block height: %w", err)
	}

	if currentHeight <= dbHeight {
		return nil // No new blocks
	}

	log.Printf("Found %d new blocks (DB: %d, Network: %d)", currentHeight-dbHeight, dbHeight, currentHeight)

	// Ingest new blocks
	for height := dbHeight + 1; height <= currentHeight; height++ {
		if err := i.ingestBlock(height); err != nil {
			return fmt.Errorf("failed to ingest block at height %d: %w", height, err)
		}
		log.Printf("Ingested new block at height %d", height)
	}

	return nil
}

func (i *Ingester) ingestBlockRange(startHeight, endHeight int64) error {
	for height := startHeight; height <= endHeight; height++ {
		if err := i.ingestBlock(height); err != nil {
			return err
		}
	}
	return nil
}

func (i *Ingester) ingestBlock(height int64) error {
	hash, err := i.rpcClient.GetBlockHash(height)
	if err != nil {
		return fmt.Errorf("failed to get block hash for height %d: %w", height, err)
	}

	// Check if block already exists
	exists, err := i.database.BlockExists(hash)
	if err != nil {
		return fmt.Errorf("failed to check if block exists: %w", err)
	}
	if exists {
		return nil // Block already ingested
	}

	block, txIDs, err := i.rpcClient.GetBlockWithTransactions(hash)
	if err != nil {
		return fmt.Errorf("failed to get block %s: %w", hash, err)
	}

	if err := i.database.InsertBlock(block); err != nil {
		return fmt.Errorf("failed to insert block %s: %w", hash, err)
	}

	// Ingest transactions for this block
	if err := i.ingestBlockTransactions(block, txIDs); err != nil {
		return fmt.Errorf("failed to ingest transactions for block %s: %w", hash, err)
	}

	return nil
}

func (i *Ingester) ingestBlockTransactions(block *models.Block, txIDs []string) error {
	totalTxs := len(txIDs)
	if totalTxs > 1 {
		log.Printf("Processing %d transactions in block %d", totalTxs, block.Height)
	}
	
	for idx, txID := range txIDs {
		if totalTxs > 10 && (idx+1)%10 == 0 {
			log.Printf("Block %d: processed %d/%d transactions", block.Height, idx+1, totalTxs)
		}
		
		tx, err := i.rpcClient.GetTransaction(txID, block.Hash)
		if err != nil {
			return fmt.Errorf("failed to get transaction %s: %w", txID, err)
		}

		// Set block-specific fields
		tx.BlockHash = block.Hash
		tx.Height = block.Height
		tx.Time = block.Timestamp

		if err := i.database.InsertTransaction(tx); err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", txID, err)
		}
	}
	
	if totalTxs > 1 {
		log.Printf("Completed processing %d transactions in block %d", totalTxs, block.Height)
	}
	
	return nil
}

func (i *Ingester) GetStats() (map[string]interface{}, error) {
	blockCount, err := i.database.GetBlockCount()
	if err != nil {
		return nil, err
	}

	latestHeight, err := i.database.GetLatestBlockHeight()
	if err != nil {
		return nil, err
	}

	networkHeight, err := i.rpcClient.GetBlockCount()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"blocks_in_db":      blockCount,
		"latest_db_height":  latestHeight,
		"network_height":    networkHeight,
		"blocks_behind":     networkHeight - latestHeight,
		"sync_percentage":   float64(latestHeight) / float64(networkHeight) * 100,
	}, nil
}