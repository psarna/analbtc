package rpc

import (
	"fmt"
	"scrapbtc/pkg/models"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

type Client struct {
	client *rpcclient.Client
}

func NewClient(host, user, pass string) (*Client, error) {
	connCfg := &rpcclient.ConnConfig{
		Host:         host,
		User:         user,
		Pass:         pass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Close() {
	c.client.Shutdown()
}

func (c *Client) GetBestBlockHeight() (int64, error) {
	count, err := c.client.GetBlockCount()
	if err != nil {
		return 0, fmt.Errorf("failed to get block count: %w", err)
	}
	return count, nil
}

func (c *Client) GetBlockHashByHeight(height int64) (string, error) {
	hash, err := c.client.GetBlockHash(height)
	if err != nil {
		return "", fmt.Errorf("failed to get block hash for height %d: %w", height, err)
	}
	return hash.String(), nil
}

func (c *Client) GetBlockWithTransactions(hash string) (*models.Block, []*models.Transaction, error) {
	blockHash, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid block hash %s: %w", hash, err)
	}
	
	// Get block with full transaction details (verbosity level 2)
	blockVerboseTx, err := c.client.GetBlockVerboseTx(blockHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block %s: %w", hash, err)
	}

	block := &models.Block{
		Hash:              blockVerboseTx.Hash,
		Height:            blockVerboseTx.Height,
		Timestamp:         time.Unix(blockVerboseTx.Time, 0),
		Size:              blockVerboseTx.Size,
		Weight:            blockVerboseTx.Weight,
		TxCount:           len(blockVerboseTx.RawTx),
		PreviousBlockHash: blockVerboseTx.PreviousHash,
		MerkleRoot:        blockVerboseTx.MerkleRoot,
		Nonce:             blockVerboseTx.Nonce,
		Bits:              blockVerboseTx.Bits,
		Difficulty:        blockVerboseTx.Difficulty,
		ProcessedAt:       time.Now(),
	}

	var transactions []*models.Transaction
	blockTime := time.Unix(blockVerboseTx.Time, 0)
	processedAt := time.Now()

	for _, rawTx := range blockVerboseTx.RawTx {
		inputValue := int64(0)
		outputValue := int64(0)
		fee := int64(0)

		for _, vout := range rawTx.Vout {
			outputValue += int64(vout.Value * 100000000)
		}

		if !isCoinbase(&rawTx) {
			for _, vin := range rawTx.Vin {
				if vin.Txid != "" {
					// For now, skip input value calculation to avoid additional RPC calls
					// This would require the previous transaction data
					// inputValue += getPrevOutputValue(vin.Txid, vin.Vout)
				}
			}
			fee = inputValue - outputValue // Will be 0 for now
		}

		tx := &models.Transaction{
			Txid:        rawTx.Txid,
			BlockHash:   hash,
			BlockHeight: blockVerboseTx.Height,
			Size:        int32(rawTx.Size),
			VSize:       int32(rawTx.Vsize),
			Weight:      int32(rawTx.Weight),
			Fee:         fee,
			InputCount:  len(rawTx.Vin),
			OutputCount: len(rawTx.Vout),
			InputValue:  inputValue,
			OutputValue: outputValue,
			Timestamp:   blockTime,
			ProcessedAt: processedAt,
		}

		transactions = append(transactions, tx)
	}

	return block, transactions, nil
}

// Deprecated: Use GetBlockWithTransactions instead
func (c *Client) GetTransactionsByBlock(blockHash string) ([]*models.Transaction, error) {
	_, transactions, err := c.GetBlockWithTransactions(blockHash)
	return transactions, err
}

func isCoinbase(tx *btcjson.TxRawResult) bool {
	return len(tx.Vin) == 1 && tx.Vin[0].Txid == ""
}