package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/mr1hm/disaster-alerts-bot/internal/bot"
	"github.com/mr1hm/disaster-alerts-bot/internal/config"
)

func main() {
	// Configure slog to use local time with JSON output
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Local().Format("2006-01-02T15:04:05.000-07:00"))
			}
			return a
		},
	})))

	_ = godotenv.Load() // ignore error if .env doesn't exist

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Token == "" {
		slog.Error("DISCORD_TOKEN is required")
		os.Exit(1)
	}
	if cfg.ChannelID == "" {
		slog.Error("DISCORD_CHANNEL_ID is required")
		os.Exit(1)
	}

	b, err := bot.New(cfg)
	if err != nil {
		slog.Error("Failed to create bot", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("Shutting down...")
		cancel()
	}()

	if err := b.Start(ctx); err != nil {
		slog.Error("Bot error", "error", err)
		os.Exit(1)
	}

	b.Stop()
	slog.Info("Bot stopped")
}
