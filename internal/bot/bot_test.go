package bot

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ewohltman/discordgo-mock/mockchannel"
	"github.com/ewohltman/discordgo-mock/mockconstants"
	"github.com/ewohltman/discordgo-mock/mockguild"
	"github.com/ewohltman/discordgo-mock/mockrest"
	"github.com/ewohltman/discordgo-mock/mocksession"
	"github.com/ewohltman/discordgo-mock/mockstate"
	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"go.uber.org/goleak"

	"github.com/mr1hm/disaster-alerts-bot/internal/config"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func newMockSession(t *testing.T, channelID string) *discordgo.Session {
	t.Helper()

	channel := mockchannel.New(
		mockchannel.WithID(channelID),
		mockchannel.WithGuildID(mockconstants.TestGuild),
		mockchannel.WithName("disaster-alerts"),
		mockchannel.WithType(discordgo.ChannelTypeGuildText),
	)

	guild := mockguild.New(
		mockguild.WithID(mockconstants.TestGuild),
		mockguild.WithName("Test Server"),
		mockguild.WithChannels(channel),
	)

	state, err := mockstate.New(mockstate.WithGuilds(guild))
	if err != nil {
		t.Fatalf("failed to create mock state: %v", err)
	}

	session, err := mocksession.New(
		mocksession.WithState(state),
		mocksession.WithClient(&http.Client{Transport: mockrest.NewTransport(state)}),
	)
	if err != nil {
		t.Fatalf("failed to create mock session: %v", err)
	}

	return session
}

func TestBot_PostDisaster(t *testing.T) {
	channelID := mockconstants.TestChannel

	session := newMockSession(t, channelID)

	b := &Bot{
		config: &config.Config{
			ChannelID: channelID,
		},
		session: session,
		posted:  make(map[string]bool),
	}

	disaster := &disastersv1.Disaster{
		Id:         "test-123",
		Title:      "M 6.5 - Near Tokyo, Japan",
		Type:       disastersv1.DisasterType_EARTHQUAKE,
		Magnitude:  6.5,
		Latitude:   35.6762,
		Longitude:  139.6503,
		AlertLevel: disastersv1.AlertLevel_ORANGE,
		Source:     "GDACS",
		Timestamp:  time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC).Unix(),
	}

	err := b.postDisaster(disaster)
	if err != nil {
		t.Fatalf("postDisaster() error = %v", err)
	}

	// Verify message was added to state
	channel, err := session.State.Channel(channelID)
	if err != nil {
		t.Fatalf("failed to get channel: %v", err)
	}

	if len(channel.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(channel.Messages))
	}

	if channel.Messages[0].Content == "" {
		t.Error("message content is empty")
	}
}

func TestBot_PostDisaster_MarksAsPosted(t *testing.T) {
	channelID := mockconstants.TestChannel

	session := newMockSession(t, channelID)

	b := &Bot{
		config: &config.Config{
			ChannelID: channelID,
		},
		session: session,
		posted:  make(map[string]bool),
	}

	disaster := &disastersv1.Disaster{
		Id:        "test-456",
		Title:     "Test Disaster",
		Type:      disastersv1.DisasterType_FLOOD,
		Magnitude: 0,
		Source:    "GDACS",
		Timestamp: time.Now().Unix(),
	}

	// Post and mark
	if err := b.postDisaster(disaster); err != nil {
		t.Fatalf("postDisaster() error = %v", err)
	}
	b.markPosted(disaster.Id)

	// Verify it's marked
	if !b.isPosted(disaster.Id) {
		t.Error("disaster should be marked as posted")
	}

	// Post again - should not add another message
	if err := b.postDisaster(disaster); err != nil {
		t.Fatalf("second postDisaster() error = %v", err)
	}

	channel, _ := session.State.Channel(channelID)
	// Note: postDisaster doesn't check isPosted - that's done in streamDisasters
	// So we expect 2 messages here (the check is at a higher level)
	if len(channel.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(channel.Messages))
	}
}

func TestBot_PostedTracking(t *testing.T) {
	b := &Bot{
		posted: make(map[string]bool),
	}

	if b.isPosted("test-1") {
		t.Error("isPosted(test-1) = true, want false")
	}

	b.markPosted("test-1")

	if !b.isPosted("test-1") {
		t.Error("isPosted(test-1) = false, want true")
	}

	if b.isPosted("test-2") {
		t.Error("isPosted(test-2) = true, want false")
	}
}

func TestBot_PostedTracking_Concurrent(t *testing.T) {
	b := &Bot{
		posted: make(map[string]bool),
	}

	var wg sync.WaitGroup
	ids := []string{"id-1", "id-2", "id-3", "id-4", "id-5"}

	// Concurrent writes
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			b.markPosted(id)
		}(id)
	}

	// Concurrent reads while writing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, id := range ids {
				_ = b.isPosted(id)
			}
		}()
	}

	wg.Wait()

	// Verify all were marked
	for _, id := range ids {
		if !b.isPosted(id) {
			t.Errorf("isPosted(%s) = false, want true", id)
		}
	}
}

func TestFormatDisasterMessage(t *testing.T) {
	disaster := &disastersv1.Disaster{
		Id:         "test-123",
		Title:      "M 6.5 - Near Tokyo, Japan",
		Type:       disastersv1.DisasterType_EARTHQUAKE,
		Magnitude:  6.5,
		Latitude:   35.6762,
		Longitude:  139.6503,
		AlertLevel: disastersv1.AlertLevel_ORANGE,
		Source:     "GDACS",
		Timestamp:  time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC).Unix(),
		Country:    "Japan",
		Population: "1.2 million in affected area",
		ReportUrl:  "https://example.com/report/123",
	}

	msg := formatDisasterMessage(disaster)

	checks := []struct {
		name  string
		value string
	}{
		{"header with emoji and type", "🟠 **EARTHQUAKE**"},
		{"title", "M 6.5 - Near Tokyo, Japan"},
		{"population", "1.2 million in affected area"},
		{"latitude", "35.6762"},
		{"magnitude", "6.5"},
		{"source", "GDACS"},
		{"alert emoji", "🟠"},
		{"Discord timestamp", "<t:"},
		{"report URL", "https://example.com/report/123"},
	}

	for _, check := range checks {
		if !contains(msg, check.value) {
			t.Errorf("message missing %s: %q", check.name, check.value)
		}
	}
}

func TestBot_ShouldPost(t *testing.T) {
	b := &Bot{
		config: &config.Config{
			MinMagnitude: 5.0,
			AlertLevel:   disastersv1.AlertLevel_ORANGE,
		},
	}

	tests := []struct {
		name     string
		disaster *disastersv1.Disaster
		want     bool
	}{
		{
			name: "earthquake above threshold",
			disaster: &disastersv1.Disaster{
				Type:      disastersv1.DisasterType_EARTHQUAKE,
				Magnitude: 6.0,
			},
			want: true,
		},
		{
			name: "earthquake below threshold",
			disaster: &disastersv1.Disaster{
				Type:      disastersv1.DisasterType_EARTHQUAKE,
				Magnitude: 4.5,
			},
			want: false,
		},
		{
			name: "earthquake at threshold",
			disaster: &disastersv1.Disaster{
				Type:      disastersv1.DisasterType_EARTHQUAKE,
				Magnitude: 5.0,
			},
			want: true,
		},
		{
			name: "flood with orange alert",
			disaster: &disastersv1.Disaster{
				Type:       disastersv1.DisasterType_FLOOD,
				AlertLevel: disastersv1.AlertLevel_ORANGE,
			},
			want: true,
		},
		{
			name: "flood with red alert",
			disaster: &disastersv1.Disaster{
				Type:       disastersv1.DisasterType_FLOOD,
				AlertLevel: disastersv1.AlertLevel_RED,
			},
			want: true,
		},
		{
			name: "flood with green alert",
			disaster: &disastersv1.Disaster{
				Type:       disastersv1.DisasterType_FLOOD,
				AlertLevel: disastersv1.AlertLevel_GREEN,
			},
			want: false,
		},
		{
			name: "cyclone with unknown alert",
			disaster: &disastersv1.Disaster{
				Type:       disastersv1.DisasterType_CYCLONE,
				AlertLevel: disastersv1.AlertLevel_UNKNOWN,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.shouldPost(tt.disaster)
			if got != tt.want {
				t.Errorf("shouldPost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAlertLevel(t *testing.T) {
	tests := []struct {
		level disastersv1.AlertLevel
		want  string
	}{
		{disastersv1.AlertLevel_GREEN, "🟢 Minor impact, localized"},
		{disastersv1.AlertLevel_ORANGE, "🟠 Moderate impact, may need international attention"},
		{disastersv1.AlertLevel_RED, "🔴 Severe impact, likely needs international humanitarian aid"},
	}

	for _, tt := range tests {
		got := formatAlertLevel(tt.level)
		if got != tt.want {
			t.Errorf("formatAlertLevel(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
