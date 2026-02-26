package jsonlog

import (
	"encoding/json"
	"io"
	"log"
	"time"
)

type Logger struct {
	base *log.Logger
}

func New(w io.Writer) *Logger {
	return &Logger{base: log.New(w, "", 0)} // no prefix; we emit JSON ourselves
}

func (l *Logger) Info(msg string, fields map[string]any) {
	l.emit("INFO", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]any) {
	l.emit("ERROR", msg, fields)
}

func (l *Logger) emit(level, msg string, fields map[string]any) {
	m := make(map[string]any, 6+len(fields))
	m["ts"] = time.Now().UTC().Format(time.RFC3339Nano)
	m["level"] = level
	m["msg"] = msg
	for k, v := range fields {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	l.base.Print(string(b))
}
