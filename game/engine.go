package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type PublicState struct {
	Gold         string         `json:"gold"`
	Diamonds     int            `json:"diamonds"`
	ClickLevel   int            `json:"clickLevel"`
	ClickPower   string         `json:"clickPower"`
	ClickCost    string         `json:"clickCost"`
	CPS          string         `json:"cps"`
	Facilities   []FacilityView `json:"facilities"`
	FacilityDefs []FacilityDef  `json:"facilityDefs"`
	ServerTime   int64          `json:"serverTime"`
}

type FacilityView struct {
	ID      int    `json:"id"`
	Owned   int    `json:"owned"`
	Enhance int    `json:"enhance"`
	Cost    string `json:"cost"`
	CPS     string `json:"cps"`
	UnitCPS string `json:"unitCps"`
}

type facilityDerived struct {
	cost        *Amount
	unitCPS     *Amount
	cps         *Amount
	costText    string
	unitCPSText string
	cpsText     string
}

type Engine struct {
	mu             sync.Mutex
	gold           *Amount
	diamonds       int
	clickLevel     int
	clickPower     *Amount
	clickCost      *Amount
	clickPowerText string
	clickCostText  string
	facilities     []FacilityState
	derived        []facilityDerived
	totalCPS       *Amount
	totalCPSText   string
	lastTick       time.Time
	store          *Store
	hub            *Hub
	rng            *rand.Rand
	dirty          bool
}

func NewEngine(store *Store) (*Engine, error) {
	p, err := store.Load()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	e := &Engine{
		gold:       amount(p.Gold),
		diamonds:   p.Diamonds,
		clickLevel: p.ClickLevel,
		facilities: normalizeFacilities(p.Facilities),
		lastTick:   now,
		store:      store,
		rng:        rand.New(rand.NewSource(now.UnixNano())),
	}
	e.initializeDerived()
	e.dirty = p.Gold != e.gold.String()
	if p.UpdatedAt > 0 {
		seconds := now.Unix() - p.UpdatedAt
		if seconds > 0 && seconds < 86400*7 {
			offline := zeroAmount().Mul(e.totalCPS, amountInt(seconds))
			e.gold.Add(e.gold, offline)
			if offline.Sign() > 0 {
				e.dirty = true
			}
		}
	}
	return e, nil
}

func (e *Engine) SetHub(h *Hub) { e.hub = h }
func (e *Engine) Start()        { go e.loop() }

func (e *Engine) loop() {
	tick := time.NewTicker(100 * time.Millisecond)
	push := time.NewTicker(time.Second)
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
	elapsed := now.Sub(e.lastTick)
	e.lastTick = now
	if elapsed > 0 && e.totalCPS.Sign() > 0 {
		gain := zeroAmount().Quo(zeroAmount().Mul(e.totalCPS, amountInt(elapsed.Nanoseconds())), amountInt(int64(time.Second)))
		e.gold.Add(e.gold, gain)
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
		Gold:       e.gold.String(),
		Diamonds:   e.diamonds,
		ClickLevel: e.clickLevel,
		Facilities: append([]FacilityState(nil), e.facilities...),
	}
	e.dirty = false
	e.mu.Unlock()
	if err := e.store.Save(p); err != nil {
		e.mu.Lock()
		e.dirty = true
		e.mu.Unlock()
	}
}

func (e *Engine) ForceSave() {
	e.mu.Lock()
	e.dirty = true
	e.mu.Unlock()
	e.persist()
}

func (e *Engine) initializeDerived() {
	e.clickPower = ClickPower(e.clickLevel)
	e.clickCost = ClickUpgradeCost(e.clickLevel)
	e.clickPowerText = e.clickPower.String()
	e.clickCostText = e.clickCost.String()
	e.derived = make([]facilityDerived, len(FacilityDefs))
	e.totalCPS = zeroAmount()
	for i, def := range FacilityDefs {
		st := e.facilities[i]
		unit := FacilityUnitCPS(def, st.Enhance)
		cps := zeroAmount().Mul(unit, amountInt(int64(st.Owned)))
		cost := FacilityCost(def, st.Owned)
		e.derived[i] = facilityDerived{
			cost:        cost,
			unitCPS:     unit,
			cps:         cps,
			costText:    cost.String(),
			unitCPSText: unit.String(),
			cpsText:     cps.String(),
		}
		e.totalCPS.Add(e.totalCPS, cps)
	}
	e.totalCPSText = e.totalCPS.String()
}

func (e *Engine) refreshFacilityDerived(idx int) {
	def := FacilityDefs[idx]
	st := e.facilities[idx]
	unit := FacilityUnitCPS(def, st.Enhance)
	cps := zeroAmount().Mul(unit, amountInt(int64(st.Owned)))
	cost := FacilityCost(def, st.Owned)
	e.derived[idx] = facilityDerived{
		cost:        cost,
		unitCPS:     unit,
		cps:         cps,
		costText:    cost.String(),
		unitCPSText: unit.String(),
		cpsText:     cps.String(),
	}
	e.totalCPS = zeroAmount()
	for i := range e.derived {
		e.totalCPS.Add(e.totalCPS, e.derived[i].cps)
	}
	e.totalCPSText = e.totalCPS.String()
}

func (e *Engine) Snapshot() PublicState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snapshotLocked()
}

func (e *Engine) snapshotLocked() PublicState {
	views := make([]FacilityView, len(FacilityDefs))
	for i, def := range FacilityDefs {
		st := e.facilities[i]
		derived := e.derived[i]
		views[i] = FacilityView{
			ID:      def.ID,
			Owned:   st.Owned,
			Enhance: st.Enhance,
			Cost:    derived.costText,
			CPS:     derived.cpsText,
			UnitCPS: derived.unitCPSText,
		}
	}
	return PublicState{
		Gold:         e.gold.String(),
		Diamonds:     e.diamonds,
		ClickLevel:   e.clickLevel,
		ClickPower:   e.clickPowerText,
		ClickCost:    e.clickCostText,
		CPS:          e.totalCPSText,
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
	gain := zeroAmount().Mul(e.clickPower, amountInt(int64(n)))
	e.gold.Add(e.gold, gain)
	diamondsGot := 0
	for i := 0; i < n; i++ {
		if e.rng.Float64() < DiamondChance {
			e.diamonds++
			diamondsGot++
		}
	}
	e.dirty = true
	state := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		b, _ := json.Marshal(map[string]any{
			"type": "click_result", "name": c.Name, "color": c.Color,
			"gain": gain.String(), "diamondsGot": diamondsGot, "state": state,
		})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doBuy(c *Client, id int) {
	e.mu.Lock()
	idx := facilityIndex(id)
	if idx < 0 {
		e.mu.Unlock()
		return
	}
	def := FacilityDefs[idx]
	derived := &e.derived[idx]
	cost := derived.cost
	if e.gold.Cmp(cost) < 0 {
		e.mu.Unlock()
		e.sendError(c, "金币不够，继续打工吧")
		return
	}
	e.gold.Sub(e.gold, cost)
	e.facilities[idx].Owned++
	e.refreshFacilityDerived(idx)
	e.dirty = true
	owned := e.facilities[idx].Owned
	state := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 购买了 %s（×%d）", c.Name, def.Name, owned), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": state})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doUpgradeClick(c *Client) {
	e.mu.Lock()
	cost := e.clickCost
	if e.gold.Cmp(cost) < 0 {
		e.mu.Unlock()
		e.sendError(c, "金币不够升级点击")
		return
	}
	e.gold.Sub(e.gold, cost)
	e.clickLevel++
	e.clickPower = ClickPower(e.clickLevel)
	e.clickCost = ClickUpgradeCost(e.clickLevel)
	e.clickPowerText = e.clickPower.String()
	e.clickCostText = e.clickCost.String()
	e.dirty = true
	power := e.clickPowerText
	state := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 升级了点击收益 → %s/次", c.Name, power), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": state})
		e.hub.Broadcast(b)
	}
}

func (e *Engine) doEnhance(c *Client, id int) {
	e.mu.Lock()
	idx := facilityIndex(id)
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
	e.refreshFacilityDerived(idx)
	e.dirty = true
	enhance := e.facilities[idx].Enhance
	name := FacilityDefs[idx].Name
	state := e.snapshotLocked()
	e.mu.Unlock()

	if e.hub != nil {
		e.hub.Notify(fmt.Sprintf("%s 用钻石强化了 %s（+%d）", c.Name, name, enhance), c.Name, c.Color)
		b, _ := json.Marshal(map[string]any{"type": "state", "state": state})
		e.hub.Broadcast(b)
	}
}

func facilityIndex(id int) int {
	for i, def := range FacilityDefs {
		if def.ID == id {
			return i
		}
	}
	return -1
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

func (e *Engine) RecentChat(limit int) ([]ChatRow, error) { return e.store.RecentChat(limit) }

func (e *Engine) sendError(c *Client, message string) {
	b, _ := json.Marshal(map[string]any{"type": "error", "text": message})
	select {
	case c.send <- b:
	default:
	}
}
