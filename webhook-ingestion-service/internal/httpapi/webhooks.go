package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"webhook-ingestion-service/internal/httpapi/webhookauth"
	"webhook-ingestion-service/internal/model"
	"webhook-ingestion-service/internal/task"
)

func WebhookProviderHandler(secret string, now func() time.Time, svc *task.Service) http.HandlerFunc {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	return func(w http.ResponseWriter, r *http.Request) {
		eventID := strings.TrimSpace(r.Header.Get("X-Event-Id"))
		if eventID == "" {
			writeError(w, http.StatusBadRequest, "missing X-Event-Id")
			return
		}

		body, err := readBody(r, maxBodyBytes)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		err = webhookauth.Verify(webhookauth.Input{
			Secret:          secret,
			TimestampHeader: r.Header.Get("X-Event-Timestamp"),
			SignatureHeader: r.Header.Get("X-Signature"),
			Body:            body,
			Now:             now(),
		})
		if err != nil {
			switch {
			case errors.Is(err, webhookauth.ErrInvalidTimestamp),
				errors.Is(err, webhookauth.ErrTimestampOutsideWindow):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				writeError(w, http.StatusUnauthorized, err.Error())
			}
			return
		}

		created, err := svc.IngestWebhook(r.Context(), eventID, body)
		if err != nil {
			if errors.Is(err, task.ErrInvalidEvent) {
				writeError(w, http.StatusBadRequest, "invalid event payload")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Webhook-friendly: always 202 if accepted (even if duplicate)
		// If you prefer: if !created { return 409 }
		_ = created
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetEventHandler(svc *task.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/events/")
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}

		ev, err := svc.GetEvent(r.Context(), id)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				writeError(w, http.StatusNotFound, "event not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, ev)
	}
}
