package protocol

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cmmarslender/go-chia-lib/pkg/config"
	"github.com/cmmarslender/go-chia-lib/pkg/streamable"
	"github.com/gorilla/websocket"
)

// Connection represents a connection with a peer and enables communication
type Connection struct {
	chiaConfig *config.ChiaConfig

	peerIP      *net.IP
	peerPort    uint16
	peerKeyPair *tls.Certificate
	peerDialer  *websocket.Dialer

	handler PeerResponseHandlerFunc

	conn *websocket.Conn
}

// PeerResponseHandlerFunc is a function that will be called when a response is returned from a peer
type PeerResponseHandlerFunc func()

// NewConnection creates a new connection object with the specified peer
func NewConnection(ip *net.IP, handler PeerResponseHandlerFunc) (*Connection, error) {
	cfg, err := config.GetChiaConfig()
	if err != nil {
		return nil, err
	}

	c := &Connection{
		chiaConfig: cfg,
		peerIP:     ip,
		peerPort:   cfg.FullNode.Port,
		handler:    handler,
	}

	err = c.loadKeyPair()
	if err != nil {
		return nil, err
	}

	// Generate the websocket dialer
	err = c.generateDialer()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Connection) loadKeyPair() error {
	var err error

	c.peerKeyPair, err = c.chiaConfig.FullNode.SSL.LoadPublicKeyPair()
	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) generateDialer() error {
	if c.peerDialer == nil {
		c.peerDialer = &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{*c.peerKeyPair},
				InsecureSkipVerify: true,
			},
		}
	}

	return nil
}

// ensureConnection ensures there is an open websocket connection
func (c *Connection) ensureConnection() error {
	if c.conn == nil {
		u := url.URL{Scheme: "wss", Host: fmt.Sprintf("%s:%d", c.peerIP.String(), c.peerPort), Path: "/ws"}
		var err error
		c.conn, _, err = c.peerDialer.Dial(u.String(), nil)
		if err != nil {
			return err
		}

		// Handshake
		handshake := &streamable.Handshake{
			NetworkID:       "mainnet", // @TODO Get the proper network ID
			ProtocolVersion: streamable.ProtocolVersion,
			SoftwareVersion: "1.2.11",
			ServerPort:      c.peerPort,
			NodeType:        streamable.NodeTypeFullNode, // I guess we're a full node
			Capabilities: []streamable.Capability{
				{
					Capability: streamable.CapabilityTypeBase,
					Value:      "1",
				},
			},
		}

		return c.Do(handshake)
	}

	return nil
}

// Do sends a request over the websocket
func (c *Connection) Do(data interface{}) error {
	err := c.ensureConnection()
	if err != nil {
		return err
	}

	encodedData, err := streamable.Marshal(data)
	if err != nil {
		return err
	}

	msg := &streamable.Message{
		ProtocolMessageType: streamable.ProtocolMessageTypeHandshake,
		Data:                encodedData,
	}

	msgBytes, err := streamable.Marshal(msg)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.BinaryMessage, msgBytes)
}
