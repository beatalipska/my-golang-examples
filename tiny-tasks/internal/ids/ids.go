package ids

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
