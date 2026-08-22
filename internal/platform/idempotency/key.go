package idempotency

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
	"time"
)

type Generator struct{ counter atomic.Uint64 }

func (g *Generator) NewID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err == nil {
		return hex.EncodeToString(buffer)
	}
	value := g.counter.Add(1)
	return time.Now().UTC().Format("20060102150405.000000000") + "-" + format(value)
}

func format(value uint64) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	if value == 0 {
		return "0"
	}
	var buffer [13]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = digits[value%uint64(len(digits))]
		value /= uint64(len(digits))
	}
	return string(buffer[index:])
}
