package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"webhook-ingestion-service/internal/httpapi/webhookauth"
)

func TestWebhookProviderHandler_AcceptsValid(t *testing.T) {
	secret := "dev-secret"
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	body := `{"type":"payment_succeeded","data":{"x":1}}`
	tsHeader := strconvI64(now.Unix())
	sig := webhookauth.SignHex(secret, tsHeader, []byte(body))

	h := WebhookProviderHandler(secret, func() time.Time { return now })

	req := httptest.NewRequest(http.MethodPost, "/webhooks/provider", strings.NewReader(body))
	req.Header.Set("X-Event-Id", "evt_123")
	req.Header.Set("X-Event-Timestamp", tsHeader)
	req.Header.Set("X-Signature", sig)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWebhookProviderHandler_MissingEventID(t *testing.T) {
	h := WebhookProviderHandler("dev-secret", func() time.Time { return time.Now().UTC() })

	req := httptest.NewRequest(http.MethodPost, "/webhooks/provider", strings.NewReader(`{}`))
	req.Header.Set("X-Event-Timestamp", strconvI64(time.Now().Unix()))
	req.Header.Set("X-Signature", "00")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWebhookProviderHandler_InvalidSignature(t *testing.T) {
	secret := "dev-secret"
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	body := `{"type":"payment_succeeded"}`
	tsHeader := strconvI64(now.Unix())
	badSig := webhookauth.SignHex("WRONG", tsHeader, []byte(body))

	h := WebhookProviderHandler(secret, func() time.Time { return now })

	req := httptest.NewRequest(http.MethodPost, "/webhooks/provider", strings.NewReader(body))
	req.Header.Set("X-Event-Id", "evt_123")
	req.Header.Set("X-Event-Timestamp", tsHeader)
	req.Header.Set("X-Signature", badSig)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWebhookProviderHandler_TimestampOutsideWindow(t *testing.T) {
	secret := "dev-secret"
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	body := `{"type":"payment_succeeded"}`
	old := now.Add(-(webhookauth.Window + time.Second))
	tsHeader := strconvI64(old.Unix())
	sig := webhookauth.SignHex(secret, tsHeader, []byte(body))

	h := WebhookProviderHandler(secret, func() time.Time { return now })

	req := httptest.NewRequest(http.MethodPost, "/webhooks/provider", strings.NewReader(body))
	req.Header.Set("X-Event-Id", "evt_123")
	req.Header.Set("X-Event-Timestamp", tsHeader)
	req.Header.Set("X-Signature", sig)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// tiny helper (avoid importing strconv everywhere in this snippet)
func strconvI64(v int64) string { return strconv.FormatInt(v, 10) }
