package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF4500"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF4500")).
			Padding(0, 1)
)

func termWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

// rawWrite writes s to stdout, replacing \n with \r\n for raw-mode compatibility.
func rawWrite(s string) {
	os.Stdout.WriteString(strings.ReplaceAll(s, "\n", "\r\n")) //nolint:errcheck
}

// PrintOverlay renders a styled response box inline in the terminal.
// title is the command name (e.g. "suggest"), response is the LLM output.
func PrintOverlay(title, response string) {
	width := termWidth() - 4
	if width < 40 {
		width = 40
	}
	if width > 120 {
		width = 120
	}

	header := titleStyle.Render("redterm") + " " + labelStyle.Render(title)
	content := strings.TrimSpace(response)
	box := boxStyle.Width(width).Render(header + "\n\n" + content)

	rawWrite("\r\n" + box + "\r\n")
}

// PrintThinking shows a brief status line while the LLM call is in flight.
func PrintThinking() {
	rawWrite("\r\n" + labelStyle.Render("[redterm] querying model...") + "\r\n")
}

// PrintBanner prints the startup banner before raw mode is active (uses normal \n).
// sessionID is the 4-char hex identifier for this redterm instance.
func PrintBanner(sessionID string) {
	width := termWidth() - 4
	if width > 72 {
		width = 72
	}
	if width < 40 {
		width = 40
	}
	content := titleStyle.Render("redterm") +
		labelStyle.Render("  AI-powered red team terminal") + "  " +
		titleStyle.Render("[RT:"+sessionID+"]") + "\n" +
		labelStyle.Render("Ctrl+G → command bar  ·  /help for commands  ·  /exit to quit")
	fmt.Println(boxStyle.Width(width).Render(content))
}

// PrintError prints an inline error message using the overlay style.
func PrintError(msg string) {
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	rawWrite("\r\n" + titleStyle.Render("redterm") + " " + errStyle.Render("error: "+msg) + "\r\n")
}
