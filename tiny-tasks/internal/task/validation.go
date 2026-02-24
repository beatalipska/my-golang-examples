package task

import "strings"

func ValidateTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if len(trimmed) < 3 {
		return "", ErrInvalidTitle
	}
	return trimmed, nil
}
