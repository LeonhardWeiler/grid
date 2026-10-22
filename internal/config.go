package internal

import (
	"time"
)

const (
	GridSize         = 512
	CooldownMs       = 5000
	MaxMessageSize   = 1024
	MaxEvents        = 100_000
	AutosaveInterval = 15 * time.Second
)

var AllowedColors = map[string]bool{
	"#ffffff": true,
	"#000000": true,
	"#ff0000": true,
	"#00ff00": true,
	"#0000ff": true,
	"#ffff00": true,
}
