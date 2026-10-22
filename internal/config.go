package internal

import (
	_ "embed"
	"encoding/json"
	"time"
)

const (
	CooldownMs       = 5000
	MaxMessageSize   = 1024
	MaxEvents        = 100_000
	AutosaveInterval = 15 * time.Second
)

//go:embed protocol/config.json
var protocolConfig []byte

type Color struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type ProtocolConfig struct {
	Protocol int     `json:"protocol"`
	GridSize int     `json:"gridSize"`
	Colors   []Color `json:"colors"`
}

var Protocol = func() ProtocolConfig {
	var config ProtocolConfig
	if err := json.Unmarshal(protocolConfig, &config); err != nil || config.GridSize <= 0 || len(config.Colors) == 0 {
		panic("invalid embedded protocol configuration")
	}
	return config
}()

var GridSize = Protocol.GridSize
var AllowedColors = func() map[string]bool {
	colors := make(map[string]bool)
	for _, c := range Protocol.Colors {
		colors[c.Value] = true
	}
	return colors
}()
