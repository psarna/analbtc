# ScrapBTC - Bitcoin Blockchain Data Scraper

A fast, concurrent Bitcoin blockchain scraper that extracts block and transaction data from Bitcoin Core RPC and stores it in DuckDB for investment analysis.

## Features

- **Concurrent Processing**: Configurable worker pool for fast data ingestion
- **Smart Resume**: Automatically skips already processed blocks
- **Real-time Progress**: Beautiful terminal UI with progress tracking
- **DuckDB Storage**: Efficient analytical database for complex queries
- **Investment Ready**: Schema designed for financial analysis

## Prerequisites

- Bitcoin Core node running locally with RPC enabled
- Go 1.24+ installed

## Usage

```bash
# Basic usage (processes last year of blocks)
./scrapbtc --user <rpc_user> --pass <rpc_pass>

# Custom date range
./scrapbtc --user <rpc_user> --pass <rpc_pass> --from 2024-01-01 --to 2024-12-31

# Custom settings
./scrapbtc \
  --user <rpc_user> \
  --pass <rpc_pass> \
  --host localhost:8332 \
  --database my_bitcoin_data.db \
  --workers 20 \
  --from 2023-01-01
```

## Command Line Options

- `--user`, `-u`: Bitcoin RPC username (required)
- `--pass`, `-p`: Bitcoin RPC password (required)  
- `--host`, `-H`: Bitcoin RPC host and port (default: localhost:8332)
- `--database`, `-d`: DuckDB database file path (default: bitcoin_data.db)
- `--from`, `-f`: Start date YYYY-MM-DD (default: 1 year ago)
- `--to`, `-t`: End date YYYY-MM-DD (default: today)
- `--workers`, `-w`: Number of concurrent workers (default: 10)

## Database Schema

The scraper creates the following tables:

- `blocks`: Block headers and metadata
- `transactions`: Transaction summaries with fees and values
- `processing_status`: Tracks which blocks have been processed

## Building

```bash
go build -o scrapbtc
```

## Example Bitcoin Core Configuration

Add to your `bitcoin.conf`:

```
server=1
rpcuser=your_username
rpcpassword=your_secure_password
rpcbind=127.0.0.1
rpcport=8332
```