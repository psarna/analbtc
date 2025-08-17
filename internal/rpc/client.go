package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"analbtc/pkg/models"
)

type Client struct {
	url        string
	httpClient *http.Client
}

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type RPCResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type BlockInfo struct {
	Hash              string    `json:"hash"`
	Confirmations     int64     `json:"confirmations"`
	Size              int64     `json:"size"`
	StrippedSize      int64     `json:"strippedsize"`
	Weight            int64     `json:"weight"`
	Height            int64     `json:"height"`
	Version           int32     `json:"version"`
	VersionHex        string    `json:"versionHex"`
	MerkleRoot        string    `json:"merkleroot"`
	Tx                []string  `json:"tx"`
	Time              int64     `json:"time"`
	MedianTime        int64     `json:"mediantime"`
	Nonce             uint32    `json:"nonce"`
	Bits              string    `json:"bits"`
	Difficulty        float64   `json:"difficulty"`
	Chainwork         string    `json:"chainwork"`
	PreviousBlockHash string    `json:"previousblockhash"`
	NextBlockHash     string    `json:"nextblockhash"`
}

type TransactionInfo struct {
	TxID     string `json:"txid"`
	Hash     string `json:"hash"`
	Version  int32  `json:"version"`
	Size     int64  `json:"size"`
	VSize    int64  `json:"vsize"`
	Weight   int64  `json:"weight"`
	Locktime uint32 `json:"locktime"`
	Vin      []VIn  `json:"vin"`
	Vout     []VOut `json:"vout"`
	Fee      int64  `json:"fee,omitempty"`
}

type VIn struct {
	TxID      string    `json:"txid"`
	Vout      uint32    `json:"vout"`
	ScriptSig ScriptSig `json:"scriptSig"`
	Sequence  uint32    `json:"sequence"`
}

type VOut struct {
	Value        float64      `json:"value"`
	N            uint32       `json:"n"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

type ScriptPubKey struct {
	Asm       string   `json:"asm"`
	Hex       string   `json:"hex"`
	ReqSigs   int      `json:"reqSigs,omitempty"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
	Address   string   `json:"address,omitempty"`
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) call(method string, params []interface{}) (*RPCResponse, error) {
	request := RPCRequest{
		JSONRPC: "1.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return &rpcResp, nil
}

func (c *Client) GetBlockCount() (int64, error) {
	resp, err := c.call("getblockcount", []interface{}{})
	if err != nil {
		return 0, err
	}

	var count int64
	if err := json.Unmarshal(resp.Result, &count); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block count: %w", err)
	}

	return count, nil
}

func (c *Client) GetBlockHash(height int64) (string, error) {
	resp, err := c.call("getblockhash", []interface{}{height})
	if err != nil {
		return "", err
	}

	var hash string
	if err := json.Unmarshal(resp.Result, &hash); err != nil {
		return "", fmt.Errorf("failed to unmarshal block hash: %w", err)
	}

	return hash, nil
}

func (c *Client) GetBlock(hash string) (*models.Block, error) {
	resp, err := c.call("getblock", []interface{}{hash, 1})
	if err != nil {
		return nil, err
	}

	var blockInfo BlockInfo
	if err := json.Unmarshal(resp.Result, &blockInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &models.Block{
		Hash:              blockInfo.Hash,
		Height:            blockInfo.Height,
		Timestamp:         time.Unix(blockInfo.Time, 0),
		Size:              blockInfo.Size,
		Weight:            blockInfo.Weight,
		TransactionCount:  int64(len(blockInfo.Tx)),
		Difficulty:        blockInfo.Difficulty,
		Bits:              blockInfo.Bits,
		Nonce:             blockInfo.Nonce,
		Version:           blockInfo.Version,
		MerkleRoot:        blockInfo.MerkleRoot,
		PreviousBlockHash: blockInfo.PreviousBlockHash,
		NextBlockHash:     blockInfo.NextBlockHash,
		Confirmations:     blockInfo.Confirmations,
		StrippedSize:      blockInfo.StrippedSize,
		MedianTime:        time.Unix(blockInfo.MedianTime, 0),
	}, nil
}

func (c *Client) GetBlockWithTransactions(hash string) (*models.Block, []string, error) {
	resp, err := c.call("getblock", []interface{}{hash, 1})
	if err != nil {
		return nil, nil, err
	}

	var blockInfo BlockInfo
	if err := json.Unmarshal(resp.Result, &blockInfo); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	block := &models.Block{
		Hash:              blockInfo.Hash,
		Height:            blockInfo.Height,
		Timestamp:         time.Unix(blockInfo.Time, 0),
		Size:              blockInfo.Size,
		Weight:            blockInfo.Weight,
		TransactionCount:  int64(len(blockInfo.Tx)),
		Difficulty:        blockInfo.Difficulty,
		Bits:              blockInfo.Bits,
		Nonce:             blockInfo.Nonce,
		Version:           blockInfo.Version,
		MerkleRoot:        blockInfo.MerkleRoot,
		PreviousBlockHash: blockInfo.PreviousBlockHash,
		NextBlockHash:     blockInfo.NextBlockHash,
		Confirmations:     blockInfo.Confirmations,
		StrippedSize:      blockInfo.StrippedSize,
		MedianTime:        time.Unix(blockInfo.MedianTime, 0),
	}

	return block, blockInfo.Tx, nil
}

func (c *Client) GetTransaction(txid, blockHash string) (*models.Transaction, error) {
	resp, err := c.call("getrawtransaction", []interface{}{txid, true, blockHash})
	if err != nil {
		return nil, err
	}

	var txInfo TransactionInfo
	if err := json.Unmarshal(resp.Result, &txInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	var inputValue, outputValue int64
	for _, vout := range txInfo.Vout {
		outputValue += int64(vout.Value * 100000000) // Convert BTC to satoshis
	}

	return &models.Transaction{
		TxID:        txInfo.TxID,
		Hash:        txInfo.Hash,
		Size:        txInfo.Size,
		VSize:       txInfo.VSize,
		Weight:      txInfo.Weight,
		Version:     txInfo.Version,
		Locktime:    txInfo.Locktime,
		InputCount:  len(txInfo.Vin),
		OutputCount: len(txInfo.Vout),
		InputValue:  inputValue,
		OutputValue: outputValue,
	}, nil
}

func (c *Client) GetBlockHeightByTime(timestamp time.Time) (int64, error) {
	count, err := c.GetBlockCount()
	if err != nil {
		return 0, err
	}

	// Binary search to find the block closest to the timestamp
	low, high := int64(0), count
	for low < high {
		mid := (low + high) / 2
		hash, err := c.GetBlockHash(mid)
		if err != nil {
			return 0, err
		}

		block, err := c.GetBlock(hash)
		if err != nil {
			return 0, err
		}

		if block.Timestamp.Before(timestamp) {
			low = mid + 1
		} else {
			high = mid
		}
	}

	return low, nil
}