package ui

import (
	"fmt"
	"os"
	"strings"

	"redterm/internal/config"
)

// ReadPrompt displays prompt and reads a line from stdin in raw mode.
// Returns the trimmed input, or "" if cancelled (Escape/Ctrl+C/Ctrl+D).
func ReadPrompt(prompt string) string {
	os.Stdout.WriteString(prompt) //nolint:errcheck

	var buf []byte
	b := make([]byte, 1)

	for {
		if _, err := os.Stdin.Read(b); err != nil {
			return ""
		}
		switch b[0] {
		case 0x1b, 3, 4:
			os.Stdout.WriteString("\r\n") //nolint:errcheck
			return ""
		case 13, 10:
			os.Stdout.WriteString("\r\n") //nolint:errcheck
			return strings.TrimSpace(string(buf))
		case 21:
			for range buf {
				os.Stdout.WriteString("\b \b") //nolint:errcheck
			}
			buf = buf[:0]
		case 127, 8:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				os.Stdout.WriteString("\b \b") //nolint:errcheck
			}
		default:
			if b[0] >= 32 {
				buf = append(buf, b[0])
				os.Stdout.Write(b) //nolint:errcheck
			}
		}
	}
}

// ReadEngagement prompts for each engagement field, updating eng in place.
// Empty input keeps the current value.
func ReadEngagement(eng *config.Engagement) {
	rawWrite("\r\n\x1b[38;5;208m[RT engage]\x1b[0m Set engagement context (Enter to keep, Esc to cancel):\r\n")

	type field struct {
		ptr  *string
		name string
	}
	fields := []field{
		{&eng.Scope, "Scope"},
		{&eng.Type, "Type (internal/external/assumed-breach/purple)"},
		{&eng.Objective, "Objective"},
		{&eng.Notes, "Notes"},
	}

	for _, f := range fields {
		prompt := fmt.Sprintf("  %-42s [%s]: ", f.name, *f.ptr)
		val := ReadPrompt(prompt)
		if val != "" {
			*f.ptr = val
		}
	}

	rawWrite("\x1b[38;5;208m[RT engage]\x1b[0m Engagement context updated.\r\n\r\n")
}

// ReadSelection displays a single-keypress prompt after /suggest and returns the
// chosen index (1-based), or 0 if the operator skips (Enter/Escape/Ctrl+C).
// count must be between 1 and 3.
func ReadSelection(count int) int {
	var keys string
	for i := 1; i <= count; i++ {
		keys += fmt.Sprintf(" [%d]", i)
	}
	prompt := "\r\n" + labelStyle.Render("run"+keys+" or Enter to skip:") + " "
	os.Stdout.WriteString(prompt) //nolint:errcheck

	b := make([]byte, 1)
	for {
		if _, err := os.Stdin.Read(b); err != nil {
			return 0
		}
		switch b[0] {
		case 0x1b, 3, 4, 13, 10:
			os.Stdout.WriteString("\r\n") //nolint:errcheck
			return 0
		default:
			if b[0] >= '1' && b[0] <= byte('0'+count) {
				os.Stdout.WriteString(string(b) + "\r\n") //nolint:errcheck
				return int(b[0] - '0')
			}
		}
	}
}

// ReadCommand displays an inline prompt and reads a slash command from stdin.
// The terminal must already be in raw mode. Returns the trimmed command string
// (e.g. "/suggest" or "/context /etc/passwd"), or "" if cancelled.
func ReadCommand() string {
	prompt := labelStyle.Render("redterm") + " " + titleStyle.Render(">") + " "
	// \r\n required in raw mode
	os.Stdout.WriteString("\r\n" + prompt) //nolint:errcheck

	var buf []byte
	b := make([]byte, 1)

	for {
		if _, err := os.Stdin.Read(b); err != nil {
			return ""
		}
		switch b[0] {
		case 0x1b: // Escape — exit command bar
			os.Stdout.WriteString("\r\n") //nolint:errcheck
			return ""
		case 3: // Ctrl+C — exit command bar
			os.Stdout.WriteString("^C\r\n") //nolint:errcheck
			return ""
		case 4: // Ctrl+D — exit command bar
			return ""
		case 13, 10: // Enter (\r or \n) — empty input exits the bar
			os.Stdout.WriteString("\r\n") //nolint:errcheck
			return strings.TrimSpace(string(buf))
		case 21: // Ctrl+U — kill line
			for range buf {
				os.Stdout.WriteString("\b \b") //nolint:errcheck
			}
			buf = buf[:0]
		case 127, 8: // Backspace / DEL
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				os.Stdout.WriteString("\b \b") //nolint:errcheck
			}
		default:
			if b[0] >= 32 { // printable ASCII
				buf = append(buf, b[0])
				os.Stdout.Write(b) //nolint:errcheck
			}
		}
	}
}
