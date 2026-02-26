package httpapi

import (
	"errors"
	"net/http"

	"webhook-ingestion-service/internal/task"
)

func ProcessOnceHandler(deps task.WorkerDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claimed, err := task.ProcessOnce(r.Context(), deps)
		if err != nil {
			if errors.Is(err, task.ErrNoWork) {
				// webhook-friendly + easy for polling clients
				w.Header().Set("X-Processed", "0")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// claimed may be true but processing failed; still 500 for debug endpoint
			_ = claimed
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// processed one item
		w.Header().Set("X-Processed", "1")
		w.WriteHeader(http.StatusNoContent)
	}
}
