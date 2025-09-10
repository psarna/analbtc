package models

import "time"

type Block struct {
	Hash              string    `json:"hash"`
	Height            int64     `json:"height"`
	Timestamp         time.Time `json:"timestamp"`
	Size              int32     `json:"size"`
	Weight            int32     `json:"weight"`
	TxCount           int       `json:"tx_count"`
	PreviousBlockHash string    `json:"previous_block_hash"`
	MerkleRoot        string    `json:"merkle_root"`
	Nonce             uint32    `json:"nonce"`
	Bits              string    `json:"bits"`
	Difficulty        float64   `json:"difficulty"`
	ProcessedAt       time.Time `json:"processed_at"`
}

type Transaction struct {
	Txid        string    `json:"txid"`
	BlockHash   string    `json:"block_hash"`
	BlockHeight int64     `json:"block_height"`
	Size        int32     `json:"size"`
	VSize       int32     `json:"vsize"`
	Weight      int32     `json:"weight"`
	Fee         int64     `json:"fee"`
	InputCount  int       `json:"input_count"`
	OutputCount int       `json:"output_count"`
	InputValue  int64     `json:"input_value"`
	OutputValue int64     `json:"output_value"`
	Timestamp   time.Time `json:"timestamp"`
	ProcessedAt time.Time `json:"processed_at"`
}

type TxInput struct {
	Txid         string `json:"txid"`
	Vout         uint32 `json:"vout"`
	ScriptSig    string `json:"script_sig"`
	Sequence     uint32 `json:"sequence"`
	PrevTxid     string `json:"prev_txid"`
	PrevVout     uint32 `json:"prev_vout"`
	Value        int64  `json:"value"`
	Address      string `json:"address"`
	TxidSpending string `json:"txid_spending"`
}

type TxOutput struct {
	Txid         string `json:"txid"`
	Vout         uint32 `json:"vout"`
	Value        int64  `json:"value"`
	ScriptPubKey string `json:"script_pub_key"`
	Address      string `json:"address"`
	SpentTxid    string `json:"spent_txid"`
	SpentVout    uint32 `json:"spent_vout"`
}

type PriceData struct {
	Timestamp  time.Time `json:"timestamp"`
	Price      float64   `json:"price"`
	MarketCap  int64     `json:"market_cap"`
	Volume24h  int64     `json:"volume_24h"`
	Source     string    `json:"source"`
	FetchedAt  time.Time `json:"fetched_at"`
}