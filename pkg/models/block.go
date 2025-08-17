package models

import "time"

type Block struct {
	Hash              string    `json:"hash"`
	Height            int64     `json:"height"`
	Timestamp         time.Time `json:"timestamp"`
	Size              int64     `json:"size"`
	Weight            int64     `json:"weight"`
	TransactionCount  int64     `json:"transaction_count"`
	Difficulty        float64   `json:"difficulty"`
	Bits              string    `json:"bits"`
	Nonce             uint32    `json:"nonce"`
	Version           int32     `json:"version"`
	MerkleRoot        string    `json:"merkle_root"`
	PreviousBlockHash string    `json:"previous_block_hash"`
	NextBlockHash     string    `json:"next_block_hash"`
	Confirmations     int64     `json:"confirmations"`
	StrippedSize      int64     `json:"stripped_size"`
	MedianTime        time.Time `json:"median_time"`
}

type Transaction struct {
	TxID     string    `json:"txid"`
	Hash     string    `json:"hash"`
	BlockHash string   `json:"block_hash"`
	Height   int64     `json:"height"`
	Time     time.Time `json:"time"`
	Size     int64     `json:"size"`
	VSize    int64     `json:"vsize"`
	Weight   int64     `json:"weight"`
	Version  int32     `json:"version"`
	Locktime uint32    `json:"locktime"`
	Fee      int64     `json:"fee"`
	InputCount  int   `json:"input_count"`
	OutputCount int   `json:"output_count"`
	InputValue  int64 `json:"input_value"`
	OutputValue int64 `json:"output_value"`
}

type UTXO struct {
	TxID     string `json:"txid"`
	Vout     uint32 `json:"vout"`
	Value    int64  `json:"value"`
	ScriptPubKey string `json:"script_pubkey"`
	Address  string `json:"address"`
	Height   int64  `json:"height"`
	Spent    bool   `json:"spent"`
	SpentTxID string `json:"spent_txid,omitempty"`
	SpentHeight int64 `json:"spent_height,omitempty"`
}