package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gofiber/contrib/websocket"
)

const (
	writeWait = 10 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space = []byte{' '}
)

type Client struct {
	hub		*Hub
	conn	*websocket.Conn
	userID	string
	send 	chan ServerMessage
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		userID: userID,
		send: make(chan ServerMessage),
	}
}

// pumps messages from the websocket connection to the hub
// messages from client to hub
// read from client
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	if c.conn == nil {
		return
	}
	for {
		var message ClientMessage
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// log error
				log.Printf("unexpected error: %v", err)
			}
			break
		}
		c.hub.incoming <- message
	}
}

// pumps messages from the hub to the websocket connection
// messages from hub to the client
// write to client
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		// reads messages from the hub
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(encodeServerMessage(message))

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := range n {
				_ = i
				w.Write(newline)
				w.Write(encodeServerMessage(<-c.send))
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func encodeServerMessage(msg ServerMessage) []byte {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {

		return nil
	}
	return jsonBytes
}
