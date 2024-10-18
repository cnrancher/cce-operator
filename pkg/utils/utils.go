package utils

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"strings"
)

func PrintObject(a any) string {
	b, _ := json.MarshalIndent(a, "", "  ")
	return string(b)
}

func Parse(ref string) (namespace string, name string) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// Generates a random hexadecimal number.
func RandomHex(l int) string {
	chars := []byte("abcdef0123456789")
	big := new(big.Int).SetInt64(int64(len(chars)))
	var b strings.Builder
	for i := 0; i < l; i++ {
		i, err := rand.Int(rand.Reader, big)
		if err != nil {
			return ""
		}
		b.WriteByte(chars[i.Int64()])
	}

	return b.String()
}

type valueTypes interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 |
		~uint32 | ~uint64 | ~uintptr | ~float32 | ~float64 | ~string | ~bool |
		[]string
}

// Pointer gets the pointer of the variable.
func Pointer[T valueTypes](i T) *T {
	return &i
}

// A safe function to get the value from the pointer.
func Value[T valueTypes](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}
