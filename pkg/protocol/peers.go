package protocol

import (
	"github.com/cmmarslender/go-chia-lib/pkg/protocols"
)

// RequestPeers asks the current peer to respond with their current peer list
func (c *Connection) RequestPeers() error {
	return c.Do(protocols.ProtocolMessageTypeRequestPeers, &protocols.RequestPeers{})
}
