package protocol

import (
	"github.com/cmmarslender/go-chia-lib/pkg/config"
)

// Connection represents a connection with a peer and enables communication
type Connection struct {
	config *config.ChiaConfig
}

// NewConnection creates a new connection object with the specified peer
func NewConnection() (*Connection, error) {
	cfg, err := config.GetChiaConfig()
	if err != nil {
		return nil, err
	}

	c := &Connection{
		config: cfg,
	}

	return c, nil
}
