# analbtc - Bitcoin Analytics System

*Where deep blockchain examination meets sophisticated data science.*

A Go-based system for ingesting Bitcoin blockchain data from a local Bitcoin RPC node into a DuckDB database for thorough analytics and indicator calculation. We believe in penetrating the depths of Bitcoin's data structure to extract meaningful insights.

## Features

- **Deep Historical Ingestion**: Penetrate Bitcoin's extensive block history starting from any configurable date
- **Continuous Monitoring**: Maintain an active connection, ready to receive fresh blockchain data as it arrives
- **High-Performance Storage**: DuckDB provides the perfect backend for intensive analytical workloads
- **Direct Bitcoin Integration**: Establish intimate connections with Bitcoin Core for real-time data flow
- **Flexible Configuration**: YAML-based setup ensures smooth entry and comfortable operation

## Prerequisites

- Go 1.21 or later
- Bitcoin Core node with RPC enabled
- DuckDB (automatically handled by Go module)

## Quick Start

*Getting started is straightforward - we'll have you up and running in no time.*

1. **Clone and prepare**:
   ```bash
   git clone <repository-url>
   cd analbtc
   make init
   ```

2. **Configure your connection**:
   Edit `config/config.yaml` with your Bitcoin RPC credentials for a secure handshake:
   ```yaml
   bitcoin:
     rpc:
       host: "localhost"
       port: 8332
       user: "your_rpc_user"
       password: "your_rpc_password"
   ```

3. **Execute with confidence**:
   ```bash
   # Build the binary
   make build

   # Begin deep historical analysis followed by continuous monitoring
   make run

   # Or choose your preferred approach:
   make run-historical  # Focus on historical penetration
   make run-continuous  # Maintain active monitoring
   make run-stats      # Review your current position
   ```

## Configuration

The system is configured via `config/config.yaml`:

```yaml
bitcoin:
  rpc:
    host: "localhost"
    port: 8332
    user: "bitcoinrpc"
    password: "your_rpc_password"
    tls: false

database:
  path: "./data/analbtc.db"

ingestion:
  start_date: "2023-08-16"  # Start ingestion from this date
  batch_size: 100           # Blocks to process in each batch
  poll_interval: "30s"      # How often to check for new blocks
```

## Database Schema

The system creates three main tables:

- **blocks**: Core block information (hash, height, timestamp, difficulty, etc.)
- **transactions**: Transaction data (txid, size, fees, input/output counts, etc.)
- **utxos**: Unspent transaction outputs for address tracking

## Usage Examples

```bash
# Show current sync status
./bin/analbtc -stats

# Ingest blocks from the last year
./bin/analbtc -historical

# Monitor for new blocks (run continuously)
./bin/analbtc -continuous

# Custom config file
./bin/analbtc -config=/path/to/config.yaml -historical
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Development build with formatting
make dev-build

# Clean build artifacts
make clean
```

## Data Analysis

*The real satisfaction comes after successful ingestion - dive deep into your collected data:*

Once data is fully ingested, you can establish an intimate connection with the DuckDB database for thorough analysis:

```sql
-- Connect to the database
.open data/analbtc.db

-- Basic statistics
SELECT COUNT(*) as total_blocks, 
       MIN(height) as first_height, 
       MAX(height) as latest_height,
       MIN(timestamp) as first_time,
       MAX(timestamp) as latest_time
FROM blocks;

-- Average block size over time
SELECT DATE_TRUNC('day', timestamp) as day,
       AVG(size) as avg_block_size,
       COUNT(*) as block_count
FROM blocks
GROUP BY DATE_TRUNC('day', timestamp)
ORDER BY day;

-- Mining difficulty trend
SELECT DATE_TRUNC('week', timestamp) as week,
       AVG(difficulty) as avg_difficulty
FROM blocks
GROUP BY DATE_TRUNC('week', timestamp)
ORDER BY week;
```

## Future Enhancements

*This robust foundation opens up exciting possibilities for deeper exploration:*

- **Advanced Indicators**: Implementation of 15+ sophisticated technical indicators
- **Market Integration**: Seamlessly merge blockchain data with price movements
- **Deep Analytics**: On-chain metrics, address clustering, and transaction flow penetration
- **API Gateway**: RESTful interface for smooth data access
- **Visual Dashboard**: Real-time Bitcoin network insights with satisfying clarity

## Architecture

- `cmd/analbtc/`: Main application entry point
- `internal/config/`: Configuration management
- `internal/db/`: DuckDB database operations
- `internal/rpc/`: Bitcoin RPC client
- `pkg/models/`: Data models
- `pkg/metrics/`: Data ingestion logic

## License

[License information]
