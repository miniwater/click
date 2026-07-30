package game

import (
	"math/big"
	"strconv"
	"strings"
)

const amountPrecision = 128

type Amount struct {
	v *big.Float
}

func amount(value string) *Amount {
	v, _, err := big.ParseFloat(value, 10, amountPrecision, big.ToNearestEven)
	if err != nil || v.IsInf() {
		return zeroAmount()
	}
	return &Amount{v: v}
}

func zeroAmount() *Amount {
	return &Amount{v: new(big.Float).SetPrec(amountPrecision).SetMode(big.ToNearestEven)}
}

func amountInt(value int64) *Amount {
	return &Amount{v: new(big.Float).SetPrec(amountPrecision).SetMode(big.ToNearestEven).SetInt64(value)}
}

func (a *Amount) Clone() *Amount {
	return &Amount{v: new(big.Float).SetPrec(amountPrecision).SetMode(big.ToNearestEven).Set(a.v)}
}

func (a *Amount) Add(x, y *Amount) *Amount {
	a.v.Add(x.v, y.v)
	return a
}

func (a *Amount) Sub(x, y *Amount) *Amount {
	a.v.Sub(x.v, y.v)
	return a
}

func (a *Amount) Mul(x, y *Amount) *Amount {
	a.v.Mul(x.v, y.v)
	return a
}

func (a *Amount) Quo(x, y *Amount) *Amount {
	a.v.Quo(x.v, y.v)
	return a
}

func (a *Amount) Cmp(other *Amount) int { return a.v.Cmp(other.v) }
func (a *Amount) Sign() int             { return a.v.Sign() }

func (a *Amount) String() string {
	if a == nil || a.v == nil || a.Sign() == 0 {
		return "0"
	}
	mantissa, exponent, _ := strings.Cut(a.v.Text('e', -1), "e")
	mantissa = strings.TrimRight(strings.TrimRight(mantissa, "0"), ".")
	exp, err := strconv.Atoi(exponent)
	if err != nil {
		return mantissa + "e" + exponent
	}
	return mantissa + "e" + strconv.Itoa(exp)
}

func amountPow(base *Amount, exponent int) *Amount {
	if exponent <= 0 {
		return amountInt(1)
	}
	result := amountInt(1)
	factor := base.Clone()
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
