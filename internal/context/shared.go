package context

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// SharedContext manages cross-terminal context sharing via per-PID temp files.
// Each redterm instance writes its output to /tmp/redterm-<uid>-<pid>.ctx;
// Snapshot() reads all live sessions' files and aggregates them.
type SharedContext struct {
	sessionID string
	ctxPath   string
	ctxFile   *os.File
	finds     *findings
}

// NewSharedContext creates a shared context for this session and registers itself
// as a buffer observer so every clean line is also tagged and written to the shared file.
func NewSharedContext(buf *Buffer) (*SharedContext, error) {
	sessionID := fmt.Sprintf("%04x", rand.Intn(0x10000))
	uid := os.Getuid()
	pid := os.Getpid()
	path := fmt.Sprintf("/tmp/redterm-%d-%d.ctx", uid, pid)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("shared context file: %w", err)
	}

	sc := &SharedContext{
		sessionID: sessionID,
		ctxPath:   path,
		ctxFile:   f,
		finds:     newFindings(),
	}
	buf.AddObserver(sc.onLine)
	return sc, nil
}

// SessionID returns the 4-char hex identifier for this session.
func (sc *SharedContext) SessionID() string {
	return sc.sessionID
}

// Close removes this session's ctx file so other sessions stop including its output.
func (sc *SharedContext) Close() {
	sc.ctxFile.Close()
	os.Remove(sc.ctxPath)
}

// onLine is the buffer observer callback: parses the line for findings and writes it
// tagged to the shared ctx file. Called with mu held on buffer — must not re-lock buffer.
func (sc *SharedContext) onLine(line string) {
	sc.finds.parseLine(line)
	fmt.Fprintf(sc.ctxFile, "[RT:%s] %s\n", sc.sessionID, line) //nolint:errcheck
}

// Snapshot aggregates context from all live redterm sessions. Prepends a structured
// findings block when findings from the local session are non-empty.
// Falls back to localBuf.Snapshot() if no ctx files are found.
func (sc *SharedContext) Snapshot(localBuf *Buffer) string {
	uid := os.Getuid()
	pattern := fmt.Sprintf("/tmp/redterm-%d-*.ctx", uid)
	matches, _ := filepath.Glob(pattern)

	if len(matches) == 0 {
		return localBuf.Snapshot()
	}

	var lines []string
	for _, path := range matches {
		pid := ctxFilePID(path)
		if pid > 0 && !pidAlive(pid) {
			os.Remove(path)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				lines = append(lines, line)
			}
		}
	}

	if len(lines) > 1500 {
		lines = lines[len(lines)-1500:]
	}

	base := strings.Join(lines, "\n")
	if block := sc.finds.format(); block != "" {
		return block + "\n" + base
	}
	return base
}

func ctxFilePID(path string) int {
	base := filepath.Base(path) // "redterm-<uid>-<pid>.ctx"
	base = strings.TrimSuffix(base, ".ctx")
	parts := strings.Split(base, "-")
	if len(parts) < 3 {
		return 0
	}
	pid, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return pid
}

func pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
