package game

import "math"

// FacilityDef 设施定义（只读配置）
type FacilityDef struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	BaseCost float64 `json:"baseCost"`
	BaseCPS  float64 `json:"baseCps"`
	Desc     string  `json:"desc"`
}

// FacilityState 设施运行时状态
type FacilityState struct {
	ID      int `json:"id"`
	Owned   int `json:"owned"`
	Enhance int `json:"enhance"`
}

var FacilityDefs = []FacilityDef{
	{1, "自动点击器", "🖱️", 15, 0.1, "机械手指，替你点点点"},
	{2, "招聘实习生", "🧑‍💼", 100, 1, "免费劳动力，几乎"},
	{3, "咖啡机", "☕", 500, 4, "加班续命神器"},
	{4, "办公工位", "🖥️", 3000, 15, "一人一座，产能翻倍"},
	{5, "购买服务器", "🖧", 10000, 50, "7×24 在线搬砖"},
	{6, "外包团队", "👷", 40000, 150, "把活扔给别人干"},
	{7, "AI 代工", "🤖", 200000, 600, "让模型帮你打工"},
	{8, "分公司", "🏢", 1_000_000, 2500, "全国连锁打工人"},
	{9, "上市融资", "📈", 5_000_000, 10000, "资本的力量"},
	{10, "全球帝国", "🌍", 50_000_000, 50000, "地球村首席打工人"},
}

const (
	ClickBasePower   = 1.0
	ClickUpgradeMult = 1.5
	ClickUpgradeBase = 50.0
	CostGrowth       = 1.05
	EnhanceMult      = 1.01
	DiamondChance    = 0.01 // 1%
)

func FacilityCost(base float64, owned int) float64 {
	return base * math.Pow(CostGrowth, float64(owned))
}

func ClickUpgradeCost(level int) float64 {
	return ClickUpgradeBase * math.Pow(CostGrowth, float64(level))
}

func ClickPower(level int) float64 {
	return ClickBasePower * math.Pow(ClickUpgradeMult, float64(level))
}

func FacilityCPS(def FacilityDef, st FacilityState) float64 {
	if st.Owned <= 0 {
		return 0
	}
	return float64(st.Owned) * def.BaseCPS * math.Pow(EnhanceMult, float64(st.Enhance))
}
