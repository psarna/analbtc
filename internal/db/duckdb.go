package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
	"analbtc/pkg/models"
)

type DuckDB struct {
	db *sql.DB
}

func NewDuckDB(dbPath string) (*DuckDB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	duckDB := &DuckDB{db: db}
	if err := duckDB.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return duckDB, nil
}

func (d *DuckDB) Close() error {
	return d.db.Close()
}

func (d *DuckDB) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS blocks (
			hash VARCHAR PRIMARY KEY,
			height BIGINT UNIQUE NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			size BIGINT NOT NULL,
			weight BIGINT NOT NULL,
			transaction_count BIGINT NOT NULL,
			difficulty DOUBLE NOT NULL,
			bits VARCHAR NOT NULL,
			nonce BIGINT NOT NULL,
			version INTEGER NOT NULL,
			merkle_root VARCHAR NOT NULL,
			previous_block_hash VARCHAR,
			next_block_hash VARCHAR,
			confirmations BIGINT NOT NULL,
			stripped_size BIGINT NOT NULL,
			median_time TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			txid VARCHAR PRIMARY KEY,
			hash VARCHAR NOT NULL,
			block_hash VARCHAR NOT NULL,
			height BIGINT NOT NULL,
			time TIMESTAMPTZ NOT NULL,
			size BIGINT NOT NULL,
			vsize BIGINT NOT NULL,
			weight BIGINT NOT NULL,
			version INTEGER NOT NULL,
			locktime BIGINT NOT NULL,
			fee BIGINT NOT NULL,
			input_count INTEGER NOT NULL,
			output_count INTEGER NOT NULL,
			input_value BIGINT NOT NULL,
			output_value BIGINT NOT NULL,
			FOREIGN KEY (block_hash) REFERENCES blocks(hash)
		)`,
		`CREATE TABLE IF NOT EXISTS utxos (
			txid VARCHAR NOT NULL,
			vout INTEGER NOT NULL,
			value BIGINT NOT NULL,
			script_pubkey VARCHAR NOT NULL,
			address VARCHAR,
			height BIGINT NOT NULL,
			spent BOOLEAN DEFAULT FALSE,
			spent_txid VARCHAR,
			spent_height BIGINT,
			PRIMARY KEY (txid, vout),
			FOREIGN KEY (txid) REFERENCES transactions(txid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_height ON blocks(height)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_block_hash ON transactions(block_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_height ON transactions(height)`,
		`CREATE INDEX IF NOT EXISTS idx_utxos_address ON utxos(address)`,
		`CREATE INDEX IF NOT EXISTS idx_utxos_spent ON utxos(spent)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

func (d *DuckDB) InsertBlock(block *models.Block) error {
	query := `INSERT INTO blocks (
		hash, height, timestamp, size, weight, transaction_count, difficulty,
		bits, nonce, version, merkle_root, previous_block_hash, next_block_hash,
		confirmations, stripped_size, median_time
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query,
		block.Hash, block.Height, block.Timestamp, block.Size, block.Weight,
		block.TransactionCount, block.Difficulty, block.Bits, block.Nonce,
		block.Version, block.MerkleRoot, block.PreviousBlockHash,
		block.NextBlockHash, block.Confirmations, block.StrippedSize,
		block.MedianTime,
	)

	return err
}

func (d *DuckDB) GetLatestBlockHeight() (int64, error) {
	var height sql.NullInt64
	err := d.db.QueryRow("SELECT MAX(height) FROM blocks").Scan(&height)
	if err != nil {
		return 0, err
	}
	
	if !height.Valid {
		return 0, nil
	}
	
	return height.Int64, nil
}

func (d *DuckDB) BlockExists(hash string) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM blocks WHERE hash = ?", hash).Scan(&count)
	return count > 0, err
}

func (d *DuckDB) InsertTransaction(tx *models.Transaction) error {
	query := `INSERT INTO transactions (
		txid, hash, block_hash, height, time, size, vsize, weight, version,
		locktime, fee, input_count, output_count, input_value, output_value
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query,
		tx.TxID, tx.Hash, tx.BlockHash, tx.Height, tx.Time, tx.Size,
		tx.VSize, tx.Weight, tx.Version, tx.Locktime, tx.Fee,
		tx.InputCount, tx.OutputCount, tx.InputValue, tx.OutputValue,
	)

	return err
}

func (d *DuckDB) GetBlockCount() (int64, error) {
	var count int64
	err := d.db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&count)
	return count, err
}