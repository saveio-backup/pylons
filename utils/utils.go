package utils

import (
	"github.com/saveio/themis/cmd/utils"
	"github.com/saveio/themis/common/constants"
	"math"
	"math/big"
	"strings"
)

func FormatUSDT(amount uint64) string {
	d := constants.USDT_DECIMALS
	a := big.NewFloat(float64(amount))
	p := big.NewFloat(math.Pow(10, float64(d)))
	c := a.Quo(a, p)
	s := c.Text('f', d)
	return RemoveRightZero(s)
}

func CutPrecision(amount string) string {
	p := DecimalPortion(amount)
	l := p - utils.PRECISION_USDT
	if l < 0 {
		l = 0
	}
	return amount[:len(amount) - l]
}

func DecimalPortion(number string) int {
	tmp := strings.Split(number, ".")
	if len(tmp) <= 1 {
		return 0
	}
	return len(tmp[1])
}

func RemoveRightZero(s string) string {
	if DecimalPortion(s) == 0 {
		return s
	}
	for i := len(s)-1; i > 0; i-- {
		if string(s[len(s)-1]) == "." {
			return s[:len(s)-1]
		}
		if string(s[len(s)-1]) == "0" {
			s = s[:len(s)-1]
		}
	}
	return s
}

