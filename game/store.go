package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS game_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  gold REAL NOT NULL DEFAULT 0,
  gold_exact TEXT NOT NULL DEFAULT '0',
  diamonds INTEGER NOT NULL DEFAULT 0,
  click_level INTEGER NOT NULL DEFAULT 0,
  facilities TEXT NOT NULL DEFAULT '[]',
  updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS chat_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  color TEXT NOT NULL,
  text TEXT NOT NULL,
  created_at INTEGER NOT NULL
);
`)
	if err != nil {
		return err
	}
	// Existing installations have the original REAL column; keep it for compatibility.
	// gold_exact now stores the bounded Amount string despite its legacy column name.
	_, _ = s.db.Exec(`ALTER TABLE game_state ADD COLUMN gold_exact TEXT NOT NULL DEFAULT '0'`)
	return nil
}

type persistedState struct {
	Gold       string          `json:"gold"`
	Diamonds   int             `json:"diamonds"`
	ClickLevel int             `json:"clickLevel"`
	Facilities []FacilityState `json:"facilities"`
	UpdatedAt  int64           `json:"updatedAt"`
}

func (s *Store) Load() (*persistedState, error) {
	row := s.db.QueryRow(`SELECT gold, gold_exact, diamonds, click_level, facilities, updated_at FROM game_state WHERE id = 1`)
	var p persistedState
	var facJSON string
	var legacyGold float64
	var exactGold string
	err := row.Scan(&legacyGold, &exactGold, &p.Diamonds, &p.ClickLevel, &facJSON, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return defaultPersisted(), nil
	}
	if err != nil {
		return nil, err
	}
	p.Gold = exactGold
	if p.Gold == "0" && legacyGold != 0 {
		p.Gold = fmt.Sprintf("%.6f", legacyGold)
	}
	if err := json.Unmarshal([]byte(facJSON), &p.Facilities); err != nil {
		p.Facilities = defaultFacilities()
	}
	p.Facilities = normalizeFacilities(p.Facilities)
	return &p, nil
}

func (s *Store) Save(p *persistedState) error {
	b, err := json.Marshal(p.Facilities)
	if err != nil {
		return err
	}
	p.UpdatedAt = time.Now().Unix()
	legacyGold := 0.0
	_, err = s.db.Exec(`
INSERT INTO game_state (id, gold, gold_exact, diamonds, click_level, facilities, updated_at)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  gold=excluded.gold,
  gold_exact=excluded.gold_exact,
  diamonds=excluded.diamonds,
  click_level=excluded.click_level,
  facilities=excluded.facilities,
  updated_at=excluded.updated_at
	`, legacyGold, p.Gold, p.Diamonds, p.ClickLevel, string(b), p.UpdatedAt)
	return err
}

type ChatRow struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Text      string `json:"text"`
	CreatedAt int64  `json:"createdAt"`
}

func (s *Store) AddChat(name, color, text string) (ChatRow, error) {
	now := time.Now().Unix()
	res, err := s.db.Exec(`INSERT INTO chat_log (name, color, text, created_at) VALUES (?,?,?,?)`, name, color, text, now)
	if err != nil {
		return ChatRow{}, err
	}
	id, _ := res.LastInsertId()
	// 只保留最近 200 条
	_, _ = s.db.Exec(`DELETE FROM chat_log WHERE id NOT IN (SELECT id FROM chat_log ORDER BY id DESC LIMIT 200)`)
	return ChatRow{ID: id, Name: name, Color: color, Text: text, CreatedAt: now}, nil
}

func (s *Store) RecentChat(limit int) ([]ChatRow, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.Query(`SELECT id, name, color, text, created_at FROM chat_log ORDER BY id DESC LIMIT ` + strconv.Itoa(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ChatRow
	for rows.Next() {
		var c ChatRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Color, &c.Text, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	// 反转为时间正序
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, nil
}

func defaultFacilities() []FacilityState {
	out := make([]FacilityState, len(FacilityDefs))
	for i, d := range FacilityDefs {
		out[i] = FacilityState{ID: d.ID, Owned: 0, Enhance: 0}
	}
	return out
}

func normalizeFacilities(in []FacilityState) []FacilityState {
	m := map[int]FacilityState{}
	for _, f := range in {
		m[f.ID] = f
	}
	out := make([]FacilityState, len(FacilityDefs))
	for i, d := range FacilityDefs {
		if f, ok := m[d.ID]; ok {
			out[i] = f
		} else {
			out[i] = FacilityState{ID: d.ID}
		}
	}
	return out
}

func defaultPersisted() *persistedState {
	return &persistedState{
		Gold:       "0",
		Diamonds:   0,
		ClickLevel: 0,
		Facilities: defaultFacilities(),
		UpdatedAt:  time.Now().Unix(),
	}
}
