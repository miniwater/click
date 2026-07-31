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

type actionRate struct {
	rateWindow time.Time
	rateCount  int
	chatWindow time.Time
	chatCount  int
	lastSeen   time.Time
}

type Hub struct {
	mu           sync.RWMutex
	clients      map[*Client]bool
	admittedByIP map[string]int
	ratesByIP    map[string]*actionRate
	admitted     int

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	engine *Engine
}

func NewHub(engine *Engine) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		admittedByIP: make(map[string]int),
		ratesByIP:    make(map[string]*actionRate),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		broadcast:    make(chan []byte, 256),
		engine:       engine,
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
				"type":  "notify",
				"text":  c.Name + " 加入了打工现场",
				"color": c.Color,
				"name":  c.Name,
			})
			h.broadcastOnline()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				h.releaseLocked(c.IP)
			}
			h.mu.Unlock()
			h.broadcastJSON(map[string]any{
				"type":  "notify",
				"text":  c.Name + " 下班了",
				"color": c.Color,
				"name":  c.Name,
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

const (
	maxConnections      = 10000
	maxConnectionsPerIP = 40
)

func (h *Hub) Admit(ip string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.admitted >= maxConnections || h.admittedByIP[ip] >= maxConnectionsPerIP {
		return false
	}
	h.admitted++
	h.admittedByIP[ip]++
	return true
}

func (h *Hub) Release(ip string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.releaseLocked(ip)
}

func (h *Hub) releaseLocked(ip string) {
	if h.admittedByIP[ip] <= 0 {
		return
	}
	h.admitted--
	h.admittedByIP[ip]--
	if h.admittedByIP[ip] == 0 {
		delete(h.admittedByIP, ip)
	}
}

func (h *Hub) allowAction(ip, action string, n int) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	rate := h.ratesByIP[ip]
	if rate == nil {
		rate = &actionRate{}
		h.ratesByIP[ip] = rate
	}
	rate.lastSeen = now
	if rate.rateWindow.IsZero() || now.Sub(rate.rateWindow) >= time.Second {
		rate.rateWindow = now
		rate.rateCount = 0
	}
	if action == "chat" {
		if rate.chatWindow.IsZero() || now.Sub(rate.chatWindow) >= 10*time.Second {
			rate.chatWindow = now
			rate.chatCount = 0
		}
		if rate.chatCount >= 5 {
			return false
		}
		rate.chatCount++
	}
	cost := 1
	if action == "click" {
		cost = max(1, min(n, 20))
	}
	if rate.rateCount+cost > 40 {
		return false
	}
	rate.rateCount += cost

	// Bound memory if many source addresses touch the service over time.
	if len(h.ratesByIP) > maxConnections*2 {
		for key, entry := range h.ratesByIP {
			if now.Sub(entry.lastSeen) > 10*time.Second && h.admittedByIP[key] == 0 {
				delete(h.ratesByIP, key)
			}
		}
	}
	return true
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
		// Never let a slow/busy hub stall the game action path.
	}
}

func (h *Hub) broadcastJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.Broadcast(b)
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
