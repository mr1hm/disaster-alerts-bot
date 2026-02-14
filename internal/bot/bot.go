package bot

import (
	"context"
	"fmt"
	"log"
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

func (b *Bot) Start(ctx context.Context) error {
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("opening discord connection: %w", err)
	}

	log.Printf("Bot started, streaming from %s", b.config.GRPCAddress)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := b.streamDisasters(ctx); err != nil {
				log.Printf("Stream error: %v, reconnecting in 5s...", err)
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (b *Bot) streamDisasters(ctx context.Context) error {
	stream, err := b.client.StreamDisasters(ctx, &disastersv1.StreamDisastersRequest{})
	if err != nil {
		return fmt.Errorf("starting stream: %w", err)
	}

	for {
		disaster, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("receiving: %w", err)
		}

		if !b.shouldPost(disaster) {
			continue
		}

		if b.isPosted(disaster.Id) {
			continue
		}

		if err := b.postDisaster(disaster); err != nil {
			log.Printf("Failed to post disaster %s: %v", disaster.Id, err)
			continue
		}

		b.markPosted(disaster.Id)
		log.Printf("Posted disaster: %s - %s (mag: %.1f)", disaster.Id, disaster.Title, disaster.Magnitude)
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
	emoji := getTypeEmoji(d.Type)
	ts := time.Unix(d.Timestamp, 0).UTC()

	msg := fmt.Sprintf(`%s **%s**
📍 Location: %.4f° N, %.4f° E
📊 Magnitude: %.1f`,
		emoji,
		d.Title,
		d.Latitude,
		d.Longitude,
		d.Magnitude,
	)

	if d.AlertLevel != disastersv1.AlertLevel_UNKNOWN {
		msg += fmt.Sprintf("\n🚨 Alert: %s", d.AlertLevel.String())
	}

	msg += fmt.Sprintf(`
🕐 Time: %s
🔗 Source: %s`,
		ts.Format("2006-01-02 15:04 UTC"),
		d.Source,
	)

	return msg
}

func getTypeEmoji(t disastersv1.DisasterType) string {
	switch t {
	case disastersv1.DisasterType_EARTHQUAKE:
		return "🌍"
	case disastersv1.DisasterType_FLOOD:
		return "🌊"
	case disastersv1.DisasterType_WILDFIRE:
		return "🔥"
	case disastersv1.DisasterType_CYCLONE:
		return "🌀"
	case disastersv1.DisasterType_TSUNAMI:
		return "🌊"
	case disastersv1.DisasterType_VOLCANO:
		return "🌋"
	case disastersv1.DisasterType_DROUGHT:
		return "☀️"
	default:
		return "⚠️"
	}
}
