package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"webhook-ingestion-service/internal/httpapi/webhookauth"
)

const maxBodyBytes = 1 << 20 // 1 MiB

func WebhookProviderHandler(secret string, now func() time.Time) http.HandlerFunc {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return func(w http.ResponseWriter, r *http.Request) {
		eventID := strings.TrimSpace(r.Header.Get("X-Event-Id"))
		if eventID == "" {
			writeError(w, http.StatusBadRequest, "missing X-Event-Id")
			return
		}

		tsHeader := r.Header.Get("X-Event-Timestamp")
		sigHeader := r.Header.Get("X-Signature")

		// Read raw body (must verify signature on raw bytes)
		body, err := readBody(r, maxBodyBytes)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		err = webhookauth.Verify(webhookauth.Input{
			Secret:          secret,
			TimestampHeader: tsHeader,
			SignatureHeader: sigHeader,
			Body:            body,
			Now:             now(),
		})
		if err != nil {
			switch {
			case errors.Is(err, webhookauth.ErrInvalidTimestamp),
				errors.Is(err, webhookauth.ErrTimestampOutsideWindow):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				// invalid signature
				writeError(w, http.StatusUnauthorized, err.Error())
			}
			return
		}

		// TODO (next step): decode JSON and insert into DB (dedup)
		// For now: accept
		w.WriteHeader(http.StatusAccepted)
	}
}

func readBody(r *http.Request, limit int64) ([]byte, error) {
	defer r.Body.Close()
	lr := io.LimitReader(r.Body, limit+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, errors.New("failed to read body")
	}
	if int64(len(b)) > limit {
		return nil, errors.New("payload too large")
	}
	return b, nil
}
