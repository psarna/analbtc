package processor

import (
	"context"
	"fmt"
	"scrapbtc/internal/db"
	"scrapbtc/internal/rpc"
	"sync"
)

type WorkerPool struct {
	rpcClient  *rpc.Client
	db         *db.DB
	numWorkers int
	progress   chan ProgressUpdate
}

type ProgressUpdate struct {
	BlockHeight int64
	TxCount     int
	Status      string
	Error       error
	DebugMsg    string
}

func NewWorkerPool(rpcClient *rpc.Client, database *db.DB, numWorkers int) *WorkerPool {
	return &WorkerPool{
		rpcClient:  rpcClient,
		db:         database,
		numWorkers: numWorkers,
		progress:   make(chan ProgressUpdate, numWorkers*2),
	}
}

func (wp *WorkerPool) ProcessBlockRange(ctx context.Context, fromHeight, toHeight int64) error {
	processedBlocks, err := wp.db.GetProcessedBlocks(fromHeight, toHeight)
	if err != nil {
		return fmt.Errorf("failed to get processed blocks: %w", err)
	}

	blockHeights := make([]int64, 0)
	for height := fromHeight; height <= toHeight; height++ {
		if !processedBlocks[height] {
			blockHeights = append(blockHeights, height)
		}
	}

	if len(blockHeights) == 0 {
		wp.progress <- ProgressUpdate{Status: "All blocks already processed"}
		close(wp.progress)
		return nil
	}

	jobs := make(chan int64, len(blockHeights))
	var wg sync.WaitGroup

	for i := 0; i < wp.numWorkers; i++ {
		wg.Add(1)
		go wp.worker(ctx, jobs, &wg)
	}

	for _, height := range blockHeights {
		select {
		case jobs <- height:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		}
	}
	close(jobs)

	wg.Wait()
	close(wp.progress)
	return nil
}

func (wp *WorkerPool) worker(ctx context.Context, jobs <-chan int64, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case height, ok := <-jobs:
			if !ok {
				return
			}
			
			if err := wp.processBlock(ctx, height); err != nil {
				wp.progress <- ProgressUpdate{
					BlockHeight: height,
					Status:      "failed",
					Error:       err,
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) processBlock(ctx context.Context, height int64) error {
	wp.progress <- ProgressUpdate{
		BlockHeight: height,
		Status:      "processing",
		DebugMsg:    fmt.Sprintf("Starting to process block %d", height),
	}

	hash, err := wp.rpcClient.GetBlockHashByHeight(height)
	if err != nil {
		wp.db.MarkBlockFailed(height, err.Error())
		return fmt.Errorf("failed to get hash for block %d: %w", height, err)
	}

	if err := wp.db.MarkBlockProcessing(height, hash); err != nil {
		return fmt.Errorf("failed to mark block processing: %w", err)
	}

	block, transactions, err := wp.rpcClient.GetBlockWithTransactions(hash)
	if err != nil {
		wp.db.MarkBlockFailed(height, err.Error())
		return fmt.Errorf("failed to get block %d with transactions: %w", height, err)
	}

	if err := wp.db.InsertBlock(block); err != nil {
		wp.db.MarkBlockFailed(height, err.Error())
		return fmt.Errorf("failed to insert block %d: %w", height, err)
	}

	for i, tx := range transactions {
		if err := wp.db.InsertTransaction(tx); err != nil {
			wp.db.MarkBlockFailed(height, err.Error())
			return fmt.Errorf("failed to insert transaction %s: %w", tx.Txid, err)
		}
		
		// Progress feedback every 100 transactions for UI updates, and every 1000 for debug log
		if (i+1)%100 == 0 || i == len(transactions)-1 {
			wp.progress <- ProgressUpdate{
				BlockHeight: height,
				TxCount:     i + 1,
				Status:      "processing_transactions",
				DebugMsg:    fmt.Sprintf("Block %d: processed %d/%d transactions", height, i+1, len(transactions)),
			}
		}
		
		// Detailed debug feedback every 1000 transactions (moved from RPC layer)
		if (i+1)%1000 == 0 {
			wp.progress <- ProgressUpdate{
				BlockHeight: height,
				TxCount:     i + 1,
				Status:      "processing_transactions",
				DebugMsg:    fmt.Sprintf("Processed %d/%d transactions for block %d", i+1, len(transactions), height),
			}
		}
	}

	if err := wp.db.MarkBlockCompleted(height); err != nil {
		return fmt.Errorf("failed to mark block completed: %w", err)
	}

	wp.progress <- ProgressUpdate{
		BlockHeight: height,
		TxCount:     len(transactions),
		Status:      "completed",
		DebugMsg:    fmt.Sprintf("Completed block %d with %d transactions", height, len(transactions)),
	}

	return nil
}

func (wp *WorkerPool) GetProgressChannel() <-chan ProgressUpdate {
	return wp.progress
}