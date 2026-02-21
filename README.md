# Disaster Alerts Bot

[![CI](https://github.com/mr1hm/disaster-alerts-bot/actions/workflows/ci.yml/badge.svg)](https://github.com/mr1hm/disaster-alerts-bot/actions/workflows/ci.yml)

A Discord bot that streams real-time disaster alerts from the [go-disaster-alerts](https://github.com/mr1hm/go-disaster-alerts) gRPC API and posts them to a Discord channel.

## Features

- Real-time streaming via gRPC (no polling)
- Automatic reconnection on stream failures (max 5 retries)
- Configurable thresholds (magnitude, alert level)
- Population-based filtering (500k+ affected required for earthquakes, triggers GREEN alerts for others)
- Deduplication via API acknowledgement (persists across restarts)
- Fetches unsent disasters on startup (last 24h)
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

The bot posts disasters that meet these criteria:
- **Earthquakes**: magnitude >= 5.0 AND 500K+ affected population
- **Other disasters**: Alert Level >= ORANGE OR 500K+ affected population

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

The bot connects to the disaster alerts gRPC server and:

1. **On startup**: Fetches unsent disasters from the last 24 hours and posts them
2. **Streams**: Receives new disasters in real-time
3. **Posts**: Formats and sends alerts to the configured Discord channel
4. **Acknowledges**: Sends gRPC acknowledgement to mark disasters as sent (prevents duplicates on restart)

## Message Format

```
🔴 **EARTHQUAKE**
**TITLE:** Red earthquake alert in Indonesia (Magnitude 7.2M, Depth:10km)
**AFFECTED:** 50,000 people in affected area
**LOCATION:** 4.2200° N, 128.2600° E
**MAGNITUDE:** 7.2
**ALERT:** 🔴 Severe impact, likely needs international humanitarian aid
**TIME:** February 14, 2026 2:30 PM (localized to user's timezone)
**SOURCE:** GDACS
https://www.gdacs.org/report.aspx?eventtype=EQ&eventid=1524431
```

Note: Magnitude is only shown for earthquakes.

Alert level indicators:
- 🟢 Minor impact, localized
- 🟠 Moderate impact, may need international attention
- 🔴 Severe impact, likely needs international humanitarian aid

## License

MIT
