package base62

import (
	"fmt"
	"math"
	"strings"
)

const (
	base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	length      = uint64(len(base62Chars))
)

// Encode returns the base62 encoding or number.
func Encode(number uint64) string {
	var encodedBuilder strings.Builder
	encodedBuilder.Grow(11)
	for ; number > 0; number = number / length {
		r := number % length
		encodedBuilder.WriteByte(base62Chars[r])
	}

	return encodedBuilder.String()
}

// Decode returns the number represented by the base62 string.
func Decode(encoded string) (uint64, error) {
	var number uint64
	for i, encodedChar := range encoded {
		base62Position := strings.IndexRune(base62Chars, encodedChar)

		if base62Position == -1 {
			return uint64(base62Position), fmt.Errorf("invalid character: %s", string(encodedChar))
		}

		number += uint64(base62Position) * uint64(math.Pow(float64(length), float64(i)))
	}

	return number, nil
}
