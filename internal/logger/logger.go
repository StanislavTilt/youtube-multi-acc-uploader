package logger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp	string	`json:"timestamp"`
	Level		string	`json:"level"`
	Category	string	`json:"category,omitempty"`
	Method		string	`json:"method,omitempty"`
	Path		string	`json:"path,omitempty"`
	Status		int	`json:"status,omitempty"`
	Duration	string	`json:"duration,omitempty"`
	IP		string	`json:"ip,omitempty"`
	UserAgent	string	`json:"userAgent,omitempty"`
	Error		string	`json:"error,omitempty"`
	Message		string	`json:"message,omitempty"`
	AccountID	int64	`json:"accountId,omitempty"`
	VideoFile	string	`json:"videoFile,omitempty"`
	YoutubeID	string	`json:"youtubeId,omitempty"`
}

type logFile struct {
	mu	sync.Mutex
	dir	string
	prefix	string
	current	*os.File
	date	string
}

func (lf *logFile) rotate() error {
	today := time.Now().Format("2006-01-02")
	if lf.date == today && lf.current != nil {
		return nil
	}
	if lf.current != nil {
		lf.current.Close()
	}
	path := filepath.Join(lf.dir, lf.prefix+"-"+today+".json")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	lf.current = f
	lf.date = today
	return nil
}

func (lf *logFile) write(entry LogEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format(time.RFC3339)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	lf.mu.Lock()
	defer lf.mu.Unlock()
	lf.rotate()
	lf.current.Write(data)
	lf.current.Write([]byte("\n"))
}

func (lf *logFile) close() {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	if lf.current != nil {
		lf.current.Close()
	}
}

type Logger struct {
	dir	string
	admin	*logFile
	youtube	*logFile
}

func New(dir string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	admin := &logFile{dir: dir, prefix: "admin"}
	youtube := &logFile{dir: dir, prefix: "youtube"}
	if err := admin.rotate(); err != nil {
		return nil, err
	}
	if err := youtube.rotate(); err != nil {
		return nil, err
	}
	return &Logger{dir: dir, admin: admin, youtube: youtube}, nil
}

func (l *Logger) Close() {
	l.admin.close()
	l.youtube.close()
}

func (l *Logger) Admin(entry LogEntry) {
	entry.Category = "admin"
	l.admin.write(entry)
}

func (l *Logger) YouTube(entry LogEntry) {
	entry.Category = "youtube"
	l.youtube.write(entry)
}

func (l *Logger) Info(msg string) {
	l.Admin(LogEntry{Level: "info", Message: msg})
}

func (l *Logger) Error(msg string) {
	l.Admin(LogEntry{Level: "error", Message: msg})
}

func (l *Logger) Dir() string {
	return l.dir
}

type responseWriter struct {
	http.ResponseWriter
	status	int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (l *Logger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		if r.URL.Path == "/" || r.URL.Path == "/favicon.ico" {
			return
		}
		ext := filepath.Ext(r.URL.Path)
		if ext == ".css" || ext == ".js" || ext == ".png" || ext == ".ico" || ext == ".svg" || ext == ".woff2" {
			return
		}

		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}

		level := "info"
		if rw.status >= 400 {
			level = "warn"
		}
		if rw.status >= 500 {
			level = "error"
		}

		l.Admin(LogEntry{
			Level:		level,
			Method:		r.Method,
			Path:		r.URL.RequestURI(),
			Status:		rw.status,
			Duration:	duration.Round(time.Microsecond).String(),
			IP:		ip,
			UserAgent:	r.UserAgent(),
		})
	})
}
