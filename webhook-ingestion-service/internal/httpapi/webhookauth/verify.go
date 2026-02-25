package webhookauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTimestamp       = errors.New("invalid timestamp")
	ErrTimestampOutsideWindow = errors.New("timestamp outside allowed window")
	ErrInvalidSignature       = errors.New("invalid signature")
)

type Input struct {
	Secret          string
	TimestampHeader string
	SignatureHeader string
	Body            []byte
	Now             time.Time
}

const Window = 5 * time.Minute

func Verify(in Input) error {
	tsHeader := strings.TrimSpace(in.TimestampHeader)
	sigHeader := strings.TrimSpace(in.SignatureHeader)

	// 1) Parse timestamp
	tsInt, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return ErrInvalidTimestamp
	}
	ts := time.Unix(tsInt, 0).UTC()

	// 2) Timestamp window check (replay protection)
	now := in.Now.UTC()
	if ts.Before(now.Add(-Window)) || ts.After(now.Add(Window)) {
		return ErrTimestampOutsideWindow
	}

	// 3) Decode provided signature (hex)
	providedSig, err := hex.DecodeString(sigHeader)
	if err != nil {
		return ErrInvalidSignature
	}

	// 4) Compute expected signature for "<ts>.<body>"
	msg := make([]byte, 0, len(tsHeader)+1+len(in.Body))
	msg = append(msg, []byte(tsHeader)...)
	msg = append(msg, '.')
	msg = append(msg, in.Body...)

	mac := hmac.New(sha256.New, []byte(in.Secret))
	_, _ = mac.Write(msg)
	expectedSig := mac.Sum(nil)

	// 5) Constant-time compare
	if !hmac.Equal(providedSig, expectedSig) {
		return ErrInvalidSignature
	}
	return nil
}

// Helper for tests/tools: compute hex signature for "<ts>.<body>"
func SignHex(secret string, timestampHeader string, body []byte) string {
	msg := make([]byte, 0, len(timestampHeader)+1+len(body))
	msg = append(msg, []byte(timestampHeader)...)
	msg = append(msg, '.')
	msg = append(msg, body...)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(msg)
	return hex.EncodeToString(mac.Sum(nil))
}
