package db

import (
	"database/sql"
	"fmt"
	"scrapbtc/pkg/models"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

func (db *DB) createTables() error {
	queries := []string{
		CreateBlocksTable,
		CreateTransactionsTable,
		CreateTxInputsTable,
		CreateTxOutputsTable,
		CreateProcessingStatusTable,
		CreatePriceDataTable,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InsertBlock(block *models.Block) error {
	query := `INSERT OR IGNORE INTO blocks (
		hash, height, timestamp, size, weight, tx_count,
		previous_block_hash, merkle_root, nonce, bits, difficulty, processed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		block.Hash, block.Height, block.Timestamp, block.Size, block.Weight,
		block.TxCount, block.PreviousBlockHash, block.MerkleRoot,
		block.Nonce, block.Bits, block.Difficulty, block.ProcessedAt)

	return err
}

func (db *DB) InsertTransaction(tx *models.Transaction) error {
	query := `INSERT OR IGNORE INTO transactions (
		txid, block_hash, block_height, size, vsize, weight, fee,
		input_count, output_count, input_value, output_value, timestamp, processed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		tx.Txid, tx.BlockHash, tx.BlockHeight, tx.Size, tx.VSize, tx.Weight,
		tx.Fee, tx.InputCount, tx.OutputCount, tx.InputValue, tx.OutputValue,
		tx.Timestamp, tx.ProcessedAt)

	return err
}

func (db *DB) InsertTransactionsBatch(transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO transactions (
		txid, block_hash, block_height, size, vsize, weight, fee,
		input_count, output_count, input_value, output_value, timestamp, processed_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, txn := range transactions {
		_, err := stmt.Exec(
			txn.Txid, txn.BlockHash, txn.BlockHeight, txn.Size, txn.VSize, txn.Weight,
			txn.Fee, txn.InputCount, txn.OutputCount, txn.InputValue, txn.OutputValue,
			txn.Timestamp, txn.ProcessedAt)
		if err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", txn.Txid, err)
		}
	}

	return tx.Commit()
}

func (db *DB) GetProcessedBlocks(fromHeight, toHeight int64) (map[int64]bool, error) {
	query := `SELECT block_height FROM processing_status WHERE status = 'completed' AND block_height BETWEEN ? AND ?`
	
	rows, err := db.conn.Query(query, fromHeight, toHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	processed := make(map[int64]bool)
	for rows.Next() {
		var height int64
		if err := rows.Scan(&height); err != nil {
			return nil, err
		}
		processed[height] = true
	}

	return processed, rows.Err()
}

func (db *DB) MarkBlockProcessing(height int64, hash string) error {
	query := `INSERT OR REPLACE INTO processing_status (block_height, block_hash, status, started_at) VALUES (?, ?, 'processing', ?)`
	_, err := db.conn.Exec(query, height, hash, time.Now())
	return err
}

func (db *DB) MarkBlockCompleted(height int64) error {
	var blockHash string
	selectQuery := `SELECT block_hash FROM processing_status WHERE block_height = ? LIMIT 1`
	err := db.conn.QueryRow(selectQuery, height).Scan(&blockHash)
	if err != nil {
		return fmt.Errorf("failed to get block hash for height %d: %w", height, err)
	}
	
	query := `INSERT OR REPLACE INTO processing_status (block_height, block_hash, status, started_at, completed_at) VALUES (?, ?, 'completed', COALESCE((SELECT started_at FROM processing_status WHERE block_height = ?), ?), ?)`
	_, err = db.conn.Exec(query, height, blockHash, height, time.Now(), time.Now())
	return err
}

func (db *DB) MarkBlockFailed(height int64, errMsg string) error {
	query := `UPDATE processing_status SET status = 'failed', completed_at = ?, error_message = ? WHERE block_height = ?`
	_, err := db.conn.Exec(query, time.Now(), errMsg, height)
	return err
}

func (db *DB) GetMaxProcessedHeight() (int64, error) {
	var maxHeight sql.NullInt64
	query := `SELECT MAX(block_height) FROM processing_status WHERE status = 'completed'`
	err := db.conn.QueryRow(query).Scan(&maxHeight)
	if err != nil {
		return 0, err
	}
	if maxHeight.Valid {
		return maxHeight.Int64, nil
	}
	return 0, nil
}

func (db *DB) CreateIndexes() error {
	if _, err := db.conn.Exec(CreateAllIndexes); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	return nil
}

func (db *DB) EnableFastInserts() error {
	// DuckDB doesn't support SQLite-specific PRAGMA statements
	// DuckDB is already optimized for fast inserts by default
	return nil
}

func (db *DB) InsertPriceData(priceData *models.PriceData) error {
	query := `INSERT OR REPLACE INTO price_data (
		timestamp, price, market_cap, volume_24h, source, fetched_at
	) VALUES (?, ?, ?, ?, ?, ?)`

	_, err := db.conn.Exec(query,
		priceData.Timestamp, priceData.Price, priceData.MarketCap,
		priceData.Volume24h, priceData.Source, priceData.FetchedAt)

	return err
}

func (db *DB) InsertPriceDataBatch(priceDataSlice []*models.PriceData) error {
	if len(priceDataSlice) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO price_data (
		timestamp, price, market_cap, volume_24h, source, fetched_at
	) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, data := range priceDataSlice {
		_, err := stmt.Exec(
			data.Timestamp, data.Price, data.MarketCap,
			data.Volume24h, data.Source, data.FetchedAt)
		if err != nil {
			return fmt.Errorf("failed to insert price data: %w", err)
		}
	}

	return tx.Commit()
}