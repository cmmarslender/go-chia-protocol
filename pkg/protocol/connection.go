package protocol

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cmmarslender/go-chia-lib/pkg/config"
	"github.com/cmmarslender/go-chia-lib/pkg/protocols"
	"github.com/gorilla/websocket"
)

// Connection represents a connection with a peer and enables communication
type Connection struct {
	chiaConfig *config.ChiaConfig

	peerIP      *net.IP
	peerPort    uint16
	peerKeyPair *tls.Certificate
	peerDialer  *websocket.Dialer

	conn *websocket.Conn
}

// PeerResponseHandlerFunc is a function that will be called when a response is returned from a peer
type PeerResponseHandlerFunc func(*protocols.Message, error)

// NewConnection creates a new connection object with the specified peer
func NewConnection(ip *net.IP) (*Connection, error) {
	cfg, err := config.GetChiaConfig()
	if err != nil {
		return nil, err
	}

	c := &Connection{
		chiaConfig: cfg,
		peerIP:     ip,
		peerPort:   cfg.FullNode.Port,
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
	}

	return nil
}

func (c *Connection) Handshake() error {
	// Handshake
	handshake := &protocols.Handshake{
		NetworkID:       "mainnet", // @TODO Get the proper network ID
		ProtocolVersion: protocols.ProtocolVersion,
		SoftwareVersion: "1.2.11",
		ServerPort:      c.peerPort,
		NodeType:        protocols.NodeTypeFullNode, // I guess we're a full node
		Capabilities: []protocols.Capability{
			{
				Capability: protocols.CapabilityTypeBase,
				Value:      "1",
			},
		},
	}

	return c.Do(protocols.ProtocolMessageTypeHandshake, handshake)
}

// Do sends a request over the websocket
func (c *Connection) Do(messageType protocols.ProtocolMessageType, data interface{}) error {
	err := c.ensureConnection()
	if err != nil {
		return err
	}

	msgBytes, err := protocols.MakeMessageBytes(messageType, data)
	if err != nil {
		return err
	}

	return c.conn.WriteMessage(websocket.BinaryMessage, msgBytes)
}

// ReadSync Reads for async responses over the connection in a synchronous fashion, blocking anything else
func (c *Connection) ReadSync(handler PeerResponseHandlerFunc) error {
	for {
		_, bytes, err := c.conn.ReadMessage()
		if err != nil {
			// @TODO Handle Error
			return err

		}
		handler(protocols.DecodeMessage(bytes))
	}
}

// ReadOne reads and returns one message from the connection
func (c *Connection) ReadOne() (*protocols.Message, error) {
	_, bytes, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	return protocols.DecodeMessage(bytes)
}
