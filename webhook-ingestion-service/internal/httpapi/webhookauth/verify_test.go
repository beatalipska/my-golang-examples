package webhookauth

import (
	"strconv"
	"testing"
	"time"
)

func TestVerify_OK(t *testing.T) {
	secret := "dev-secret"
	body := []byte(`{"type":"payment_succeeded","data":{"x":1}}`)

	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	ts := now.Add(-2 * time.Minute).Unix()
	tsHeader := itoa(ts)

	sig := SignHex(secret, tsHeader, body)

	err := Verify(Input{
		Secret:          secret,
		TimestampHeader: tsHeader,
		SignatureHeader: sig,
		Body:            body,
		Now:             now,
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestVerify_InvalidTimestamp(t *testing.T) {
	err := Verify(Input{
		Secret:          "dev-secret",
		TimestampHeader: "not-a-number",
		SignatureHeader: "00",
		Body:            []byte(`{}`),
		Now:             time.Now(),
	})
	if err != ErrInvalidTimestamp {
		t.Fatalf("expected ErrInvalidTimestamp, got %v", err)
	}
}

func TestVerify_OutsideWindow_TooOld(t *testing.T) {
	secret := "dev-secret"
	body := []byte(`{"k":"v"}`)

	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	ts := now.Add(-(Window + time.Second)).Unix()
	tsHeader := itoa(ts)

	sig := SignHex(secret, tsHeader, body)

	err := Verify(Input{
		Secret:          secret,
		TimestampHeader: tsHeader,
		SignatureHeader: sig,
		Body:            body,
		Now:             now,
	})
	if err != ErrTimestampOutsideWindow {
		t.Fatalf("expected ErrTimestampOutsideWindow, got %v", err)
	}
}

func TestVerify_OutsideWindow_TooFuture(t *testing.T) {
	secret := "dev-secret"
	body := []byte(`{"k":"v"}`)

	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	ts := now.Add(Window + time.Second).Unix()
	tsHeader := itoa(ts)

	sig := SignHex(secret, tsHeader, body)

	err := Verify(Input{
		Secret:          secret,
		TimestampHeader: tsHeader,
		SignatureHeader: sig,
		Body:            body,
		Now:             now,
	})
	if err != ErrTimestampOutsideWindow {
		t.Fatalf("expected ErrTimestampOutsideWindow, got %v", err)
	}
}

func TestVerify_InvalidSignature_BadHex(t *testing.T) {
	err := Verify(Input{
		Secret:          "dev-secret",
		TimestampHeader: itoa(time.Now().Unix()),
		SignatureHeader: "not-hex!!!",
		Body:            []byte(`{}`),
		Now:             time.Now(),
	})
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerify_InvalidSignature_WrongSecret(t *testing.T) {
	secret := "dev-secret"
	body := []byte(`{"hello":"world"}`)

	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	tsHeader := itoa(now.Unix())

	sig := SignHex("WRONG-SECRET", tsHeader, body)

	err := Verify(Input{
		Secret:          secret,
		TimestampHeader: tsHeader,
		SignatureHeader: sig,
		Body:            body,
		Now:             now,
	})
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
