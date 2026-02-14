package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mr1hm/disaster-alerts-bot/internal/bot"
	"github.com/mr1hm/disaster-alerts-bot/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Token == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}
	if cfg.ChannelID == "" {
		log.Fatal("DISCORD_CHANNEL_ID is required")
	}

	b, err := bot.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	if err := b.Start(ctx); err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	b.Stop()
	log.Println("Bot stopped")
}
