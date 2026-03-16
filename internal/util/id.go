package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomHex(8))
}

func NewToken() string {
	return randomHex(24)
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
