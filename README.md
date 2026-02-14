# Disaster Alerts Bot

[![CI](https://github.com/mr1hm/disaster-alerts-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/mr1hm/disaster-alerts-bot/actions/workflows/ci.yml)

A Discord bot that streams real-time disaster alerts from the [go-disaster-alerts](https://github.com/mr1hm/go-disaster-alerts) gRPC API and posts them to a Discord channel.

## Features

- Real-time streaming via gRPC (no polling)
- Automatic reconnection on stream failures
- Configurable thresholds (magnitude, alert level)
- Deduplication of posted alerts
- Graceful shutdown on SIGINT/SIGTERM

### Coming Soon

- Slash commands (`/get {disasterID}` to fetch disaster details)

## Prerequisites

- Go 1.25+
- A running [go-disaster-alerts](https://github.com/mr1hm/go-disaster-alerts) gRPC server
- Discord bot token and channel ID

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/mr1hm/disaster-alerts-bot.git
   cd disaster-alerts-bot
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create a `.env` file:
   ```bash
   cp .env.example .env
   ```

4. Configure your environment variables (see below).

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_TOKEN` | Yes | - | Discord bot token |
| `DISCORD_CHANNEL_ID` | Yes | - | Channel ID to post alerts |
| `GRPC_ADDRESS` | No | `localhost:50051` | gRPC server address |
| `MIN_MAGNITUDE` | No | `5.0` | Minimum magnitude for earthquakes |
| `ALERT_LEVEL` | No | `ORANGE` | Minimum alert level for other disasters |

### Filtering

The bot filters disasters before posting:
- **Earthquakes**: posted if `magnitude >= MIN_MAGNITUDE`
- **Other disasters**: posted if `alert_level >= ALERT_LEVEL`

Alert levels: `GREEN` < `ORANGE` < `RED`

## Running

```bash
# Start the bot
go run ./cmd/bot

# Or build and run
go build -o bot ./cmd/bot
./bot
```

## Testing

```bash
# Run all tests with race detector
go test -race ./...
```

## Architecture

```
cmd/bot/main.go          # Entry point, signal handling
internal/
├── config/config.go     # Environment configuration
└── bot/bot.go           # Discord bot, gRPC streaming
```

The bot connects to the disaster alerts gRPC server and streams new disasters in real-time. When a disaster is received, it formats a message and posts it to the configured Discord channel.

## Message Format

```
**TITLE:** M 6.5 - Near Tokyo, Japan
**LOCATION:** 35.6762° N, 139.6503° E
**MAGNITUDE:** 6.5
**ALERT:** 🟠 Moderate impact, may need international attention
**TIME:** January 15, 2026 2:30 PM (localized to user's timezone)
**SOURCE:** USGS
```

Alert level indicators:
- 🟢 Minor impact, localized
- 🟠 Moderate impact, may need international attention
- 🔴 Severe impact, likely needs international humanitarian aid

## License

MIT
