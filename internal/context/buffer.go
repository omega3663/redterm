package context

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// ansiRE strips ANSI/VT escape sequences so LLM context is clean text.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[^[]|\x1b`)

// StripANSI removes ANSI/VT escape sequences from s.
func StripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// Observer is called with each clean (ANSI-stripped, non-empty) line added to the buffer.
// It is invoked while the buffer lock is held — observers must not call back into Buffer.
type Observer func(line string)

// Buffer is a thread-safe rolling ring buffer of terminal output lines.
type Buffer struct {
	mu        sync.Mutex
	lines     []string
	cap       int
	head      int
	size      int
	partial   strings.Builder
	observers []Observer
}

// AddObserver registers fn to be called with each new line pushed to the buffer.
func (b *Buffer) AddObserver(fn Observer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.observers = append(b.observers, fn)
}

func New(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = 500
	}
	return &Buffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

// Write implements io.Writer. Accepts raw PTY output including ANSI escape codes.
// Splits on newlines; partial lines are held until the next newline arrives.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range string(p) {
		if ch == '\n' {
			b.pushLine(b.partial.String())
			b.partial.Reset()
		} else if ch != '\r' {
			b.partial.WriteRune(ch)
		}
	}
	return len(p), nil
}

// pushLine strips ANSI codes and stores the line. Caller must hold mu.
func (b *Buffer) pushLine(line string) {
	line = ansiRE.ReplaceAllString(line, "")
	if strings.TrimSpace(line) == "" {
		return
	}
	b.lines[b.head] = line
	b.head = (b.head + 1) % b.cap
	if b.size < b.cap {
		b.size++
	}
	for _, obs := range b.observers {
		obs(line)
	}
}

// Snapshot returns all buffered lines joined with newlines, oldest first.
func (b *Buffer) Snapshot() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size == 0 {
		return ""
	}
	out := make([]string, b.size)
	start := (b.head - b.size + b.cap) % b.cap
	for i := range out {
		out[i] = b.lines[(start+i)%b.cap]
	}
	return strings.Join(out, "\n")
}

// InjectFile reads path and appends its contents to the buffer.
func (b *Buffer) InjectFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	b.InjectText(fmt.Sprintf("[file: %s]\n%s", path, string(data)))
	return nil
}

// InjectText appends arbitrary text to the buffer as individual lines.
func (b *Buffer) InjectText(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, line := range strings.Split(text, "\n") {
		b.pushLine(line)
	}
}

// Clear empties the buffer.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.size = 0
	b.head = 0
	b.partial.Reset()
}
