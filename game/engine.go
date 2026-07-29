package game

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type PublicState struct {
	Gold         float64        `json:"gold"`
	Diamonds     int            `json:"diamonds"`
	ClickLevel   int            `json:"clickLevel"`
	ClickPower   float64        `json:"clickPower"`
	ClickCost    float64       `json:"clickCost"`
	CPS          float64        `json:"cps"`
	Facilities   []FacilityView `json:"facilities"`
	FacilityDefs []FacilityDef  `json:"facilityDefs"`
	ServerTime   int64          `json:"serverTime"`
}

type FacilityView struct {
	ID      int     `json:"id"`
	Owned   int     `json:"owned"`
	Enhance int     `json:"enhance"`
	Cost    float64 `json:"cost"`
	CPS     float64 `json:"cps"`
	UnitCPS float64 `json:"unitCps"`
}

type Engine struct {
	mu         sync.Mutex
	gold       float64
	diamonds   int
	clickLevel int
	facilities []FacilityState
	lastTick   time.Time
	store      *Store
	hub        *Hub
	rng        *rand.Rand
	dirty      bool
}

func NewEngine(store *Store) (*Engine, error) {
	p, err := store.Load()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	e := &Engine{
		gold:       p.Gold,
		diamonds:   p.Diamonds,
		clickLevel: p.ClickLevel,
		facilities: normalizeFacilities(p.Facilities),
		lastTick:   now,
		store:      store,
		rng:        rand.New(rand.NewSource(now.UnixNano())),
	}
	if p.UpdatedAt > 0 {
		elapsed := now.Sub(time.Unix(p.UpdatedAt, 0)).Seconds()
		if elapsed > 0 && elapsed < 86400*7 {
			e.gold += e.calcCPSLocked() * elapsed
		}
	}
	return e, nil
}

func (e *Engine) SetHub(h *Hub) {
	e.hub = h
}

func (e *Engine) Start() {
	go e.loop()
}

func (e *Engine) loop() {
	tick := time.NewTicker(100 * time.Millisecond)
	push := time.NewTicker(1 * time.Second)
	save := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	defer push.Stop()
	defer save.Stop()
	for {
		select {
		case <-tick.C:
			e.onTick()
		case <-push.C:
			if e.hub != nil {
				e.hub.BroadcastState()
			}
		case <-save.C:
			e.persist()
		}
	}
}

func (e *Engine) onTick() {
	e.mu.Lock()
	now := time.Now()
	dt := now.Sub(e.lastTick).Seconds()
	e.lastTick = now
	cps := e.calcCPSLocked()
	if dt > 0 && cps > 0 {
		e.gold += cps * dt
		e.dirty = true
	}
	e.mu.Unlock()
}

func (e *Engine) persist() {
	e.mu.Lock()
	if !e.dirty {
		e.mu.Unlock()
		return
	}
	p := &persistedState{
		Gold:       e.gold,
		Diamonds:   e.diamonds,
		ClickLevel: e.clickLevel,
		Facilities: append([]FacilityState(nil), e.facilities...),
	}
	e.dirty = false
	e.mu.Unlock()
	_ = e.store.Save(p)
}

func (e *Engine) ForceSave() {
	e.mu.Lock()
	e.dirty = true
	e.mu.Unlock()
	e.persist()
}

func (e *Engine) calcCPSLocked() float64 {
	var total float64
	for i, def := range FacilityDefs {
		total += FacilityCPS(def, e.facilities[i])
	}
	return total
}

func (e *Engine) Snapshot() PublicState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *Engine) snapshotLocked() PublicState {
	views := make([]FacilityView, len(FacilityDefs))
	var cps float64
	for i, def := range FacilityDefs {
		st := e.facilities[i]
		unit := def.BaseCPS * math.Pow(EnhanceMult, float64(st.Enhance))
		fcps := FacilityCPS(def, st)
		cps += fcps
		views[i] = FacilityView{
			ID:      def.ID,
			Owned:   st.Owned,
			Enhance: st.Enhance,
			Cost:    FacilityCost(def.BaseCost, st.Owned),
			CPS:     fcps,
			UnitCPS: unit,
		}
	}
	return PublicState{
		Gold:         e.gold,
		Diamonds:     e.diamonds,
		ClickLevel:   e.clickLevel,
		ClickPower:   ClickPower(e.clickLevel),
		ClickCost:    ClickUpgradeCost(e.clickLevel),
		CPS:          cps,
		Facilities:   views,
		FacilityDefs: FacilityDefs,
		ServerTime:   time.Now().UnixMilli(),
	}
}

type inbound struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	Text string `json:"text"`
	N    int    `json:"n"`
}

func (e *Engine) HandleMessage(c *Client, data []byte) {
	var msg inbound
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}
	switch msg.Type {
	case "click":
		e.doClick(c, msg.N)
	case "buy":
		e.doBuy(c, msg.ID)
	case "upgrade_click":
		e.doUpgradeClick(c)
	case "enhance":
		e.doEnhance(c, msg.ID)
	case "chat":
		e.doChat(c, msg.Text)
	}
}

func (e *Engine) doClick(c *Client, n int) {
	if n <= 0 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	e.mu.Lock()
	power := ClickPower(e.clickLevel)
	gain := power * float64(n)
	e.gold += gain
	diamondsGot := 0
	for i := 0; i < n; i++ {
		if e.rng.Float64() < DiamondChance {
			e.diamonds++
			diamondsGot++
		}
	}
	e.dirty = true
	st := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		b, _ := json.Marshal(map[string]any{
			"type":        "click_result",
			"name":        c.Name,
			"color":       c.Color,
			"gain":        gain,
			"diamondsGot": diamondsGot,
			"state":       st,
		})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doBuy(c *Client, id int) {
	e.mu.Lock()
	idx := -1
	for i, d := range FacilityDefs {
		if d.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		e.mu.Unlock()
		return
	}
	def := FacilityDefs[idx]
	cost := FacilityCost(def.BaseCost, e.facilities[idx].Owned)
	if e.gold < cost {
		e.mu.Unlock()
		e.sendError(c, "金币不够，继续打工吧")
		return
	}
	e.gold -= cost
	e.facilities[idx].Owned++
	e.dirty = true
	owned := e.facilities[idx].Owned
	st := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 购买了 %s（×%d）", c.Name, def.Name, owned), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": st})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doUpgradeClick(c *Client) {
	e.mu.Lock()
	cost := ClickUpgradeCost(e.clickLevel)
	if e.gold < cost {
		e.mu.Unlock()
		e.sendError(c, "金币不够升级点击")
		return
	}
	e.gold -= cost
	e.clickLevel++
	e.dirty = true
	power := ClickPower(e.clickLevel)
	st := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 升级了点击收益 → %.2f/次", c.Name, power), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": st})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doEnhance(c *Client, id int) {
	e.mu.Lock()
	idx := -1
	for i, d := range FacilityDefs {
		if d.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		e.mu.Unlock()
		return
	}
	if e.facilities[idx].Owned <= 0 {
		e.mu.Unlock()
		e.sendError(c, "先购买该设施再强化")
		return
	}
	if e.diamonds < 1 {
		e.mu.Unlock()
		e.sendError(c, "钻石不足")
		return
	}
	e.diamonds--
	e.facilities[idx].Enhance++
	e.dirty = true
	enh := e.facilities[idx].Enhance
	name := FacilityDefs[idx].Name
	st := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 用钻石强化了 %s（+%d）", c.Name, name, enh), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": st})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doChat(c *Client, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if utf8.RuneCountInString(text) > 100 {
		text = string([]rune(text)[:100])
	}
	text = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, text)
	row, err := e.store.AddChat(c.Name, c.Color, text)
	if err != nil {
		return
	}
	if e.hub != nil {
		b, _ := json.Marshal(map[string]any{"type": "chat", "chat": row})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) RecentChat(limit int) ([]ChatRow, error) {
	return e.store.RecentChat(limit)
}

func (e *Engine) sendError(c *Client, msg string) {
	b, _ := json.Marshal(map[string]any{"type": "error", "text": msg})
	select {
	case c.send <- b:
	default:
	}
}
