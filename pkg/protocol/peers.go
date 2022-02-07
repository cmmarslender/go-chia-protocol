package protocol

// RequestPeers asks the current peer to respond with their current peer list
func (c *Connection) RequestPeers() error {
	// @TODO for now, just sending any connection at all
	return c.ensureConnection()
}
