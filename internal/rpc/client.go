package rpc

import (
	"encoding/json"
	"fmt"
	"scrapbtc/pkg/models"
	"time"

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

	// Test connection by getting blockchain info
	info, err := client.GetBlockChainInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Bitcoin RPC: %w", err)
	}
	
	fmt.Printf("Connected to Bitcoin RPC - Chain: %s, Blocks: %d\n", info.Chain, info.Blocks)
	
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
	// Try to get block with full transaction details using a raw JSON-RPC call
	// This uses verbosity level 2 which should include full transaction details
	params := []json.RawMessage{
		json.RawMessage(`"` + hash + `"`),
		json.RawMessage(`2`),
	}
	result, err := c.client.RawRequest("getblock", params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get block %s with verbosity 2: %w", hash, err)
	}

	// Parse the result manually since the btcd library doesn't support verbosity=2 properly
	var blockData struct {
		Hash              string  `json:"hash"`
		Height            int64   `json:"height"`
		Time              int64   `json:"time"`
		Size              int32   `json:"size"`
		Weight            int32   `json:"weight"`
		PreviousBlockHash string  `json:"previousblockhash"`
		MerkleRoot        string  `json:"merkleroot"`
		Nonce             uint32  `json:"nonce"`
		Bits              string  `json:"bits"`
		Difficulty        float64 `json:"difficulty"`
		Tx                []struct {
			Txid     string `json:"txid"`
			Size     int32  `json:"size"`
			VSize    int32  `json:"vsize"`
			Weight   int32  `json:"weight"`
			Vin      []struct {
				Txid string `json:"txid"`
				Vout uint32 `json:"vout"`
			} `json:"vin"`
			Vout []struct {
				Value float64 `json:"value"`
			} `json:"vout"`
		} `json:"tx"`
	}

	if err := json.Unmarshal(result, &blockData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal block data: %w", err)
	}

	block := &models.Block{
		Hash:              blockData.Hash,
		Height:            blockData.Height,
		Timestamp:         time.Unix(blockData.Time, 0),
		Size:              blockData.Size,
		Weight:            blockData.Weight,
		TxCount:           len(blockData.Tx),
		PreviousBlockHash: blockData.PreviousBlockHash,
		MerkleRoot:        blockData.MerkleRoot,
		Nonce:             blockData.Nonce,
		Bits:              blockData.Bits,
		Difficulty:        blockData.Difficulty,
		ProcessedAt:       time.Now(),
	}

	var transactions []*models.Transaction
	blockTime := time.Unix(blockData.Time, 0)
	processedAt := time.Now()

	for _, rawTx := range blockData.Tx {
		inputValue := int64(0)
		outputValue := int64(0)
		fee := int64(0)

		for _, vout := range rawTx.Vout {
			outputValue += int64(vout.Value * 100000000)
		}

		// Check if it's coinbase transaction
		isCoinbaseTx := len(rawTx.Vin) == 1 && rawTx.Vin[0].Txid == ""
		
		if !isCoinbaseTx {
			// For now, skip input value calculation to avoid additional RPC calls
			// This would require the previous transaction data
			fee = inputValue - outputValue // Will be 0 for now
		}

		tx := &models.Transaction{
			Txid:        rawTx.Txid,
			BlockHash:   hash,
			BlockHeight: blockData.Height,
			Size:        rawTx.Size,
			VSize:       rawTx.VSize,
			Weight:      rawTx.Weight,
			Fee:         fee,
			InputCount:  len(rawTx.Vin),
			OutputCount: len(rawTx.Vout),
			InputValue:  inputValue,
			OutputValue: outputValue,
			Timestamp:   blockTime,
			ProcessedAt: processedAt,
		}

		transactions = append(transactions, tx)
		
		// Progress feedback is now handled by the processor layer
	}

	return block, transactions, nil
}

// Deprecated: Use GetBlockWithTransactions instead
func (c *Client) GetTransactionsByBlock(blockHash string) ([]*models.Transaction, error) {
	_, transactions, err := c.GetBlockWithTransactions(blockHash)
	return transactions, err
}