package chat

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type incomingMessage struct {
	RoomID  uint   `json:"room_id"`
	Message string `json:"message"`
}

type Client struct {
	hub       *Hub
	service   *Service
	conn      *websocket.Conn
	send      chan []byte
	senderID  uint
	roomID    uint
	isSupport bool
}

func (c *Client) trySend(data []byte) {
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("chat: %v", err)
			}
			return
		}
		var in incomingMessage
		if err := json.Unmarshal(raw, &in); err != nil {
			in.Message = string(raw)
		}
		in.Message = strings.TrimSpace(in.Message)
		if in.Message == "" {
			continue
		}
		ctx := context.Background()
		if c.isSupport {
			if in.RoomID == 0 {
				continue
			}
			if _, err := c.service.SupportReply(ctx, in.RoomID, c.senderID, in.Message); err != nil {
				log.Printf("chat: support reply: %v", err)
			}
			continue
		}
		if _, err := c.service.UserMessage(ctx, c.roomID, c.senderID, in.Message); err != nil {
			log.Printf("chat: user message: %v", err)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
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
