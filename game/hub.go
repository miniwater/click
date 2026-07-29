package game

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	Name  string
	Color string
	IP    string
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	engine *Engine
}

func NewHub(engine *Engine) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		engine:     engine,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.sendStateTo(c)
			h.broadcastJSON(map[string]any{
				"type": "notify",
				"text": c.Name + " 加入了打工现场",
				"color": c.Color,
				"name": c.Name,
			})
			h.broadcastOnline()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()
			h.broadcastJSON(map[string]any{
				"type": "notify",
				"text": c.Name + " 下班了",
				"color": c.Color,
				"name": c.Name,
			})
			h.broadcastOnline()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// 慢客户端丢弃
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

func (h *Hub) broadcastJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.broadcast <- b
}

func (h *Hub) BroadcastState() {
	st := h.engine.Snapshot()
	h.broadcastJSON(map[string]any{
		"type":  "state",
		"state": st,
	})
}

func (h *Hub) sendStateTo(c *Client) {
	st := h.engine.Snapshot()
	chats, _ := h.engine.RecentChat(50)
	b, _ := json.Marshal(map[string]any{
		"type":   "welcome",
		"name":   c.Name,
		"color":  c.Color,
		"state":  st,
		"chats":  chats,
		"online": h.OnlineCount(),
	})
	select {
	case c.send <- b:
	default:
	}
}

func (h *Hub) broadcastOnline() {
	h.broadcastJSON(map[string]any{
		"type":   "online",
		"online": h.OnlineCount(),
	})
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	c.conn.SetReadLimit(4096)

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.hub.engine.HandleMessage(c, data)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWS(hub *Hub, conn *websocket.Conn, ip string) {
	name := MaskIP(ip)
	color := ThemeColorFromIP(ip)
	c := &Client{
		hub:   hub,
		conn:  conn,
		send:  make(chan []byte, 64),
		Name:  name,
		Color: color,
		IP:    ip,
	}
	hub.register <- c
	go c.writePump()
	c.readPump()
}

func (h *Hub) Notify(text, name, color string) {
	h.broadcastJSON(map[string]any{
		"type":  "notify",
		"text":  text,
		"name":  name,
		"color": color,
	})
}

func LogErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
