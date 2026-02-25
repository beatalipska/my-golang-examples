package httpapi

import (
	"errors"
	"io"
	"net/http"
)

const maxBodyBytes = 1 << 20 // 1 MiB

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
