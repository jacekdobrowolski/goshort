package base62

import (
	"math"
	"strings"
)

const base62Digits = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Encode(value uint64) string {
	if value == 0 {
		return "0"
	}

	base62 := make([]byte, 0)
	radix := uint64(62)

	for value > 0 {
		remainder := value % radix
		base62 = append([]byte{base62Digits[remainder]}, base62...)
		value /= radix
	}

	return string(base62)
}

func Decode(data string) uint64 {
	var decimalNumber uint64

	for i, c := range data {
		base62offset := strings.IndexByte(base62Digits, byte(c))
		if base62offset < 0 {
			panic("assert: byte c not in base62 character set")
		}

		decimalNumber += uint64(base62offset) * uint64(math.Pow(float64(62), float64(len(data)-i-1)))
	}

	return decimalNumber
}
