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

func (c *Client) GetBlockByHash(hash string) (*models.Block, error) {
	blockHash, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash %s: %w", hash, err)
	}
	
	blockVerbose, err := c.client.GetBlockVerbose(blockHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %s: %w", hash, err)
	}

	block := &models.Block{
		Hash:              blockVerbose.Hash,
		Height:            blockVerbose.Height,
		Timestamp:         time.Unix(blockVerbose.Time, 0),
		Size:              blockVerbose.Size,
		Weight:            blockVerbose.Weight,
		TxCount:           len(blockVerbose.Tx),
		PreviousBlockHash: blockVerbose.PreviousHash,
		MerkleRoot:        blockVerbose.MerkleRoot,
		Nonce:             blockVerbose.Nonce,
		Bits:              blockVerbose.Bits,
		Difficulty:        blockVerbose.Difficulty,
		ProcessedAt:       time.Now(),
	}

	return block, nil
}

func (c *Client) GetTransactionsByBlock(blockHash string) ([]*models.Transaction, error) {
	hash, err := chainhash.NewHashFromStr(blockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash %s: %w", blockHash, err)
	}
	
	blockVerbose, err := c.client.GetBlockVerbose(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %s: %w", blockHash, err)
	}

	var transactions []*models.Transaction
	blockTime := time.Unix(blockVerbose.Time, 0)
	processedAt := time.Now()

	for _, txid := range blockVerbose.Tx {
		txHash, err := chainhash.NewHashFromStr(txid)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction hash %s: %w", txid, err)
		}
		
		rawTx, err := c.client.GetRawTransactionVerbose(txHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction %s: %w", txid, err)
		}

		inputValue := int64(0)
		outputValue := int64(0)
		fee := int64(0)

		for _, vout := range rawTx.Vout {
			outputValue += int64(vout.Value * 100000000)
		}

		if !isCoinbase(rawTx) {
			for _, vin := range rawTx.Vin {
				if vin.Txid != "" {
					prevTxHash, err := chainhash.NewHashFromStr(vin.Txid)
					if err != nil {
						continue
					}
					
					prevTx, err := c.client.GetRawTransactionVerbose(prevTxHash)
					if err != nil {
						continue
					}
					if int(vin.Vout) < len(prevTx.Vout) {
						inputValue += int64(prevTx.Vout[vin.Vout].Value * 100000000)
					}
				}
			}
			fee = inputValue - outputValue
		}

		tx := &models.Transaction{
			Txid:        rawTx.Txid,
			BlockHash:   blockHash,
			BlockHeight: blockVerbose.Height,
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

	return transactions, nil
}

func isCoinbase(tx *btcjson.TxRawResult) bool {
	return len(tx.Vin) == 1 && tx.Vin[0].Txid == ""
}