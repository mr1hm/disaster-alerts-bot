package config

import (
	"os"
	"strconv"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
)

type Config struct {
	Token        string
	ChannelID    string
	GRPCAddress  string
	MinMagnitude *float64
	DisasterType disastersv1.DisasterType
	AlertLevel   disastersv1.AlertLevel
}

func Load() (*Config, error) {
	cfg := &Config{
		Token:       os.Getenv("DISCORD_TOKEN"),
		ChannelID:   os.Getenv("DISCORD_CHANNEL_ID"),
		GRPCAddress: getEnvOrDefault("GRPC_ADDRESS", "localhost:50051"),
	}

	if minMag := os.Getenv("MIN_MAGNITUDE"); minMag != "" {
		if mag, err := strconv.ParseFloat(minMag, 64); err == nil {
			cfg.MinMagnitude = &mag
		}
	}

	if dt := os.Getenv("DISASTER_TYPE"); dt != "" {
		if val, ok := disastersv1.DisasterType_value[dt]; ok {
			cfg.DisasterType = disastersv1.DisasterType(val)
		}
	}

	if al := os.Getenv("ALERT_LEVEL"); al != "" {
		if val, ok := disastersv1.AlertLevel_value[al]; ok {
			cfg.AlertLevel = disastersv1.AlertLevel(val)
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
