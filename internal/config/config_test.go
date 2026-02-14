package config

import (
	"os"
	"testing"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GRPCAddress != "localhost:50051" {
		t.Errorf("GRPCAddress = %q, want %q", cfg.GRPCAddress, "localhost:50051")
	}
	if cfg.MinMagnitude != 5.0 {
		t.Errorf("MinMagnitude = %v, want 5.0", cfg.MinMagnitude)
	}
	if cfg.AlertLevel != disastersv1.AlertLevel_ORANGE {
		t.Errorf("AlertLevel = %v, want ORANGE", cfg.AlertLevel)
	}
}

func TestLoad_EnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("DISCORD_TOKEN", "test-token")
	os.Setenv("DISCORD_CHANNEL_ID", "123456")
	os.Setenv("GRPC_ADDRESS", "localhost:9000")
	os.Setenv("MIN_MAGNITUDE", "6.0")
	os.Setenv("ALERT_LEVEL", "RED")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "test-token")
	}
	if cfg.ChannelID != "123456" {
		t.Errorf("ChannelID = %q, want %q", cfg.ChannelID, "123456")
	}
	if cfg.GRPCAddress != "localhost:9000" {
		t.Errorf("GRPCAddress = %q, want %q", cfg.GRPCAddress, "localhost:9000")
	}
	if cfg.MinMagnitude != 6.0 {
		t.Errorf("MinMagnitude = %v, want 6.0", cfg.MinMagnitude)
	}
	if cfg.AlertLevel != disastersv1.AlertLevel_RED {
		t.Errorf("AlertLevel = %v, want RED", cfg.AlertLevel)
	}
}
