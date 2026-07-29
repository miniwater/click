package game

import (
	"math/big"
	"strings"
)

type FacilityDef struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	BaseCost string `json:"baseCost"`
	BaseCPS  string `json:"baseCps"`
	Desc     string `json:"desc"`
}

type FacilityState struct {
	ID      int `json:"id"`
	Owned   int `json:"owned"`
	Enhance int `json:"enhance"`
}

var FacilityDefs = []FacilityDef{
	{1, "自动点击器", "🖱️", "15", "0.1", "机械手指，替你点点点"},
	{2, "招聘实习生", "🧑‍💼", "100", "1", "免费劳动力，几乎"},
	{3, "咖啡机", "☕", "500", "4", "加班续命神器"},
	{4, "办公工位", "🖥️", "3000", "15", "一人一座，产能翻倍"},
	{5, "购买服务器", "🖧", "10000", "50", "7×24 在线搬砖"},
	{6, "外包团队", "👷", "40000", "150", "把活扔给别人干"},
	{7, "AI 代工", "🤖", "200000", "600", "让模型帮你打工"},
	{8, "分公司", "🏢", "1000000", "2500", "全国连锁打工人"},
	{9, "上市融资", "📈", "5000000", "10000", "资本的力量"},
	{10, "全球帝国", "🌍", "50000000", "50000", "地球村首席打工人"},
	{11, "月球办事处", "🌕", "500000000", "250000", "把加班文化送上月球"},
	{12, "轨道数据中心", "🛰️", "8000000000", "2000000", "在近地轨道处理全球订单"},
	{13, "火星工业园", "🔴", "150000000000", "15000000", "跨行星流水线昼夜不停"},
	{14, "小行星矿场", "☄️", "3000000000000", "120000000", "开采太空里的第一桶金"},
	{15, "戴森云节点", "☀️", "80000000000000", "1000000000", "收集恒星能源驱动产线"},
	{16, "星际物流网", "🚀", "2500000000000000", "9000000000", "让货物穿梭于各个星系"},
	{17, "量子财务中心", "⚛️", "100000000000000000", "90000000000", "同时结算无数条时间线"},
	{18, "银河贸易联盟", "🌌", "5000000000000000000", "1000000000000", "垄断银河系的打工市场"},
	{19, "时空管理局", "⏳", "300000000000000000000", "12000000000000", "从过去和未来同时收取利润"},
	{20, "多元宇宙集团", "♾️", "25000000000000000000000", "160000000000000", "每个宇宙都有一家分公司"},
	{21, "维度折叠工厂", "🌀", "3000000000000000000000000", "2500000000000000", "把生产线折叠进额外维度"},
	{22, "因果律交易所", "🔗", "500000000000000000000000000", "45000000000000000", "在结果发生前结算利润"},
	{23, "真空能采集站", "✨", "100000000000000000000000000000", "900000000000000000", "从量子真空提取无尽能源"},
	{24, "宇宙常数实验室", "🧪", "30000000000000000000000000000000", "20000000000000000000", "微调常数提高全宇宙产能"},
	{25, "时间线复制中心", "🕰️", "10000000000000000000000000000000000", "500000000000000000000", "复制最赚钱的历史进程"},
	{26, "奇点计算集群", "⚫", "5000000000000000000000000000000000000", "15000000000000000000000", "在事件视界内完成无限计算"},
	{27, "文明孵化矩阵", "🧬", "3000000000000000000000000000000000000000", "500000000000000000000000", "批量培育会打工的星际文明"},
	{28, "宇宙重启服务", "🔄", "3000000000000000000000000000000000000000000", "20000000000000000000000000", "每次重启都保留全部利润"},
	{29, "现实编译器", "⌨️", "5000000000000000000000000000000000000000000000", "1000000000000000000000000000", "直接编译一条更富有的现实"},
	{30, "无限层级总部", "🏛️", "10000000000000000000000000000000000000000000000000", "60000000000000000000000000000", "管理没有终点的打工体系"},
}

const DiamondChance = 0.01

func decimal(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return new(big.Rat)
	}
	return r
}

func ratPow(base *big.Rat, exponent int) *big.Rat {
	if exponent <= 0 {
		return new(big.Rat).SetInt64(1)
	}
	result := new(big.Rat).SetInt64(1)
	factor := new(big.Rat).Set(base)
	for exponent > 0 {
		if exponent&1 == 1 {
			result.Mul(result, factor)
		}
		exponent >>= 1
		if exponent > 0 {
			factor.Mul(factor, factor)
		}
	}
	return result
}

func FacilityCost(def FacilityDef, owned int) *big.Rat {
	growth := decimal("1.05")
	if def.ID >= 21 {
		growth = decimal("1.25")
	}
	return new(big.Rat).Mul(decimal(def.BaseCost), ratPow(growth, owned))
}

func ClickUpgradeCost(level int) *big.Rat {
	return new(big.Rat).Mul(decimal("10"), ratPow(decimal("1.05"), level))
}

func ClickPower(level int) *big.Rat {
	return ratPow(decimal("1.05"), level)
}

func FacilityUnitCPS(def FacilityDef, enhance int) *big.Rat {
	return new(big.Rat).Mul(decimal(def.BaseCPS), ratPow(decimal("1.01"), enhance))
}

func FacilityCPS(def FacilityDef, st FacilityState) *big.Rat {
	if st.Owned <= 0 {
		return new(big.Rat)
	}
	return new(big.Rat).Mul(FacilityUnitCPS(def, st.Enhance), new(big.Rat).SetInt64(int64(st.Owned)))
}

func decimalString(value *big.Rat) string {
	if value.Sign() == 0 {
		return "0"
	}

	numerator := new(big.Int).Set(value.Num())
	negative := numerator.Sign() < 0
	numerator.Abs(numerator)
	denominator := new(big.Int).Set(value.Denom())
	two := big.NewInt(2)
	five := big.NewInt(5)
	remainder := new(big.Int)
	twos, fives := 0, 0
	for {
		remainder.Mod(denominator, two)
		if remainder.Sign() != 0 {
			break
		}
		denominator.Div(denominator, two)
		twos++
	}
	for {
		remainder.Mod(denominator, five)
		if remainder.Sign() != 0 {
			break
		}
		denominator.Div(denominator, five)
		fives++
	}
	if denominator.Cmp(big.NewInt(1)) != 0 {
		return strings.TrimRight(strings.TrimRight(value.FloatString(18), "0"), ".")
	}

	scale := twos
	if fives > scale {
		scale = fives
	}
	if twos < scale {
		numerator.Mul(numerator, new(big.Int).Exp(two, big.NewInt(int64(scale-twos)), nil))
	}
	if fives < scale {
		numerator.Mul(numerator, new(big.Int).Exp(five, big.NewInt(int64(scale-fives)), nil))
	}
	digits := numerator.String()
	if scale > 0 {
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		point := len(digits) - scale
		digits = digits[:point] + "." + digits[point:]
		digits = strings.TrimRight(strings.TrimRight(digits, "0"), ".")
	}
	if negative {
		digits = "-" + digits
	}
	s := digits
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}
