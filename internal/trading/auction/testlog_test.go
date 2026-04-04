package auction

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

var logMu sync.Mutex

const packageName = "auction"

type logEntry struct {
	Time    string `json:"time"`
	Package string `json:"package"`
	Test    string `json:"test"`
	Message string `json:"message"`
}

func logf(t *testing.T, format string, args ...any) {
	t.Helper()
	path := logFilePath()
	logMu.Lock()
	defer logMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	entry := logEntry{
		Time:    time.Now().Format(time.RFC3339Nano),
		Package: packageName,
		Test:    t.Name(),
		Message: fmt.Sprintf(format, args...),
	}
	encoded, _ := json.Marshal(entry)
	_, _ = f.Write(encoded)
	_, _ = f.Write([]byte("\n"))
}

func logSeparator(t *testing.T, label string) {
	t.Helper()
	path := logFilePath()
	logMu.Lock()
	defer logMu.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "========== %s %s (%s) ==========\n", label, t.Name(), packageName)
}

func logFilePath() string {
	if dir := os.Getenv("TEST_LOG_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
		return filepath.Join(dir, "trading-test-report.log")
	}
	root := repoRoot()
	dir := filepath.Join(root, "test-log")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, fmt.Sprintf("trading-test-report-%s.log", packageName))
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func logTest(t *testing.T) {
	t.Helper()
	logSeparator(t, "START")
	logf(t, "START")
	t.Cleanup(func() {
		logf(t, "END")
		logSeparator(t, "END")
	})
}
