package utils

import (
	"github.com/saveio/themis/cmd/utils"
	"strings"
)

func FormatUSDT(amount uint64) string {
	u := utils.FormatUsdt(amount)
	return CutPrecision(u)
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