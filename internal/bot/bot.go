package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/disaster-alerts-bot/internal/config"
)

type Bot struct {
	config  *config.Config
	session *discordgo.Session
	conn    *grpc.ClientConn
	client  disastersv1.DisasterServiceClient
	posted  map[string]bool
	mu      sync.RWMutex
}

func New(cfg *config.Config) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("creating discord session: %w", err)
	}

	conn, err := grpc.NewClient(cfg.GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connecting to grpc server: %w", err)
	}

	return &Bot{
		config:  cfg,
		session: session,
		conn:    conn,
		client:  disastersv1.NewDisasterServiceClient(conn),
		posted:  make(map[string]bool),
	}, nil
}

const maxRetries = 5

func (b *Bot) Start(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("opening discord connection: %w", err)
	}

	slog.Info("Bot started", "grpc_address", b.config.GRPCAddress)

	// Fetch and post existing disasters on startup
	if err := b.fetchInitialDisasters(ctx); err != nil {
		slog.Error("Failed to fetch initial disasters", "error", err)
		// Continue anyway - streaming will still work
	}

	retries := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			connected, err := b.streamDisasters(ctx)
			if err != nil {
				// Check if we're shutting down
				if ctx.Err() != nil {
					return nil
				}

				if connected {
					// Stream was working, reset retry count
					retries = 0
					slog.Info("Stream disconnected, reconnecting", "error", err)
				} else {
					retries++
					if retries >= maxRetries {
						return fmt.Errorf("stream failed after %d retries: %w", maxRetries, err)
					}
					slog.Error("Stream error, reconnecting", "error", err, "retry", retries, "max_retries", maxRetries)
				}
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (b *Bot) streamDisasters(ctx context.Context) (connected bool, err error) {
	stream, err := b.client.StreamDisasters(ctx, &disastersv1.StreamDisastersRequest{})
	if err != nil {
		return false, fmt.Errorf("starting stream: %w", err)
	}

	slog.Info("Connected to disaster stream")

	for {
		disaster, err := stream.Recv()
		if err != nil {
			return connected, fmt.Errorf("receiving: %w", err)
		}

		connected = true // Successfully received at least one message

		if !b.shouldPost(disaster) {
			continue
		}

		if b.isPosted(disaster.Id) {
			continue
		}

		if err := b.postDisaster(disaster); err != nil {
			slog.Error("Failed to post disaster", "id", disaster.Id, "error", err)
			continue
		}

		b.markPosted(disaster.Id)
		slog.Info("Posted disaster", "id", disaster.Id, "title", disaster.Title, "magnitude", disaster.Magnitude)
	}
}

func (b *Bot) shouldPost(d *disastersv1.Disaster) bool {
	if d.Type == disastersv1.DisasterType_EARTHQUAKE {
		return d.Magnitude >= b.config.MinMagnitude
	}
	return d.AlertLevel >= b.config.AlertLevel
}

func (b *Bot) Stop() {
	if b.session != nil {
		b.session.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
}

func (b *Bot) fetchInitialDisasters(ctx context.Context) error {
	alertLevel := b.config.AlertLevel
	resp, err := b.client.ListDisasters(ctx, &disastersv1.ListDisastersRequest{
		Limit:         50,
		MinAlertLevel: &alertLevel,
	})
	if err != nil {
		return fmt.Errorf("listing disasters: %w", err)
	}

	slog.Info("Fetched initial disasters", "count", len(resp.Disasters), "min_alert", alertLevel.String())

	for _, disaster := range resp.Disasters {
		if err := b.postDisaster(disaster); err != nil {
			slog.Error("Failed to post initial disaster", "id", disaster.Id, "error", err)
			continue
		}
		b.markPosted(disaster.Id)
	}

	slog.Info("Posted initial disasters", "count", len(resp.Disasters))
	return nil
}

func (b *Bot) postDisaster(d *disastersv1.Disaster) error {
	msg := formatDisasterMessage(d)
	_, err := b.session.ChannelMessageSend(b.config.ChannelID, msg)
	return err
}

func (b *Bot) isPosted(id string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.posted[id]
}

func (b *Bot) markPosted(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.posted[id] = true
}

func formatDisasterMessage(d *disastersv1.Disaster) string {
	// Header: 🔴 **EARTHQUAKE** - Indonesia
	alertEmoji := getAlertEmoji(d.AlertLevel)
	header := fmt.Sprintf("%s **%s**", alertEmoji, d.Type.String())
	if d.Country != "" {
		header += " - " + d.Country
	}

	lines := []string{
		header,
		fmt.Sprintf("**TITLE:** %s", d.Title),
	}

	if d.Population != "" {
		lines = append(lines, fmt.Sprintf("**AFFECTED:** %s", d.Population))
	}

	lines = append(lines,
		fmt.Sprintf("**LOCATION:** %.4f° N, %.4f° E", d.Latitude, d.Longitude),
		fmt.Sprintf("**MAGNITUDE:** %.1f", d.Magnitude),
	)

	if d.AlertLevel != disastersv1.AlertLevel_UNKNOWN {
		lines = append(lines, fmt.Sprintf("**ALERT:** %s", formatAlertLevel(d.AlertLevel)))
	}

	lines = append(lines,
		fmt.Sprintf("**TIME:** <t:%d:F>", d.Timestamp),
		fmt.Sprintf("**SOURCE:** %s", d.Source),
	)

	if d.ReportUrl != "" {
		lines = append(lines, d.ReportUrl)
	}

	return strings.Join(lines, "\n")
}

func getAlertEmoji(level disastersv1.AlertLevel) string {
	switch level {
	case disastersv1.AlertLevel_GREEN:
		return "🟢"
	case disastersv1.AlertLevel_ORANGE:
		return "🟠"
	case disastersv1.AlertLevel_RED:
		return "🔴"
	default:
		return "⚪"
	}
}

func formatAlertLevel(level disastersv1.AlertLevel) string {
	switch level {
	case disastersv1.AlertLevel_GREEN:
		return "🟢 Minor impact, localized"
	case disastersv1.AlertLevel_ORANGE:
		return "🟠 Moderate impact, may need international attention"
	case disastersv1.AlertLevel_RED:
		return "🔴 Severe impact, likely needs international humanitarian aid"
	default:
		return level.String()
	}
}

