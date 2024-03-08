package base62

import (
	"math"
	"strings"
)

const base62Digits = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Encode(n uint64) string {
	if n == 0 {
		return "0"
	}

	base62 := make([]byte, 0)
	radix := uint64(62)

	for n > 0 {
		remainder := n % radix
		base62 = append([]byte{base62Digits[remainder]}, base62...)
		n /= radix
	}

	return string(base62)
}

func Decode(s string) uint64 {
	var decimalNumber uint64

	for i, c := range s {
		decimalNumber += uint64(strings.IndexByte(base62Digits, byte(c))) * uint64(math.Pow(62, float64(len(s)-i-1)))
	}

	return decimalNumber
}
