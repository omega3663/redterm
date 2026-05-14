package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"

	"redterm/internal/config"
	redcontext "redterm/internal/context"
	"redterm/internal/intelligence"
	"redterm/internal/llm"
	"redterm/internal/ui"
)

// Run spawns shell in a PTY and drives the main I/O loop.
func Run(shell string, triggerByte byte, provider llm.Provider, buf *redcontext.Buffer, sc *redcontext.SharedContext, eng *config.Engagement) error {
	if eng == nil {
		eng = &config.Engagement{}
	}

	sessionID := sc.SessionID()
	cmd, cleanupShell := buildShellCmd(shell, sessionID)
	defer cleanupShell()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("pty start: %w", err)
	}
	defer ptmx.Close()

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			pty.InheritSize(os.Stdin, ptmx) //nolint:errcheck
		}
	}()
	pty.InheritSize(os.Stdin, ptmx) //nolint:errcheck

	// Banner is printed before raw mode so normal \n works.
	ui.PrintBanner(sessionID)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("raw mode: %w", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigterm
		term.Restore(int(os.Stdin.Fd()), oldState) //nolint:errcheck
		os.Exit(0)
	}()

	var inCmdMode atomic.Bool

	go func() {
		readBuf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(readBuf)
			if n > 0 {
				data := readBuf[:n]
				buf.Write(data) //nolint:errcheck
				if !inCmdMode.Load() {
					os.Stdout.Write(data) //nolint:errcheck
				}
			}
			if err != nil {
				return
			}
		}
	}()

	b := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(b)
		if n == 0 || err == io.EOF {
			break
		}

		if b[0] == triggerByte {
			inCmdMode.Store(true)
			for {
				cmdStr := ui.ReadCommand()
				if cmdStr == "" {
					break
				}
				exit, execCmd := handleCommand(cmdStr, provider, buf, sc, eng)
				if exit {
					inCmdMode.Store(false)
					ptmx.Write([]byte("exit\n")) //nolint:errcheck
					return cmd.Wait()
				}
				if execCmd != "" {
					inCmdMode.Store(false)
					ptmx.Write([]byte(execCmd + "\n")) //nolint:errcheck
					break
				}
			}
			inCmdMode.Store(false)
			continue
		}

		if _, err := ptmx.Write(b[:n]); err != nil {
			break
		}
	}

	signal.Stop(sigwinch)
	signal.Stop(sigterm)
	return cmd.Wait()
}

// buildShellCmd constructs the shell command with PS1 indicator for bash/zsh.
// Returns the command and a cleanup func to remove temp files.
func buildShellCmd(shell, sessionID string) (*exec.Cmd, func()) {
	base := filepath.Base(shell)
	baseEnv := append(os.Environ(),
		"TERM=xterm-256color",
		"REDTERM=1",
		"REDTERM_SESSION="+sessionID,
	)
	noop := func() {}

	switch base {
	case "bash":
		rcContent := fmt.Sprintf(
			"[ -f ~/.bashrc ] && . ~/.bashrc\n"+
				"export REDTERM=1\n"+
				"export REDTERM_SESSION=%s\n"+
				`PS1='\[\e[38;5;208m\][RT:%s]\[\e[0m\] '"$PS1"`+"\n",
			sessionID, sessionID,
		)
		rcFile := fmt.Sprintf("/tmp/redterm-rc-%s", sessionID)
		if err := os.WriteFile(rcFile, []byte(rcContent), 0600); err == nil {
			cmd := exec.Command(shell, "--rcfile", rcFile)
			cmd.Env = baseEnv
			return cmd, func() { os.Remove(rcFile) }
		}

	case "zsh":
		zdotdir := fmt.Sprintf("/tmp/redterm-zdotdir-%s", sessionID)
		if err := os.MkdirAll(zdotdir, 0700); err == nil {
			rcContent := fmt.Sprintf(
				"[ -f $HOME/.zshrc ] && source $HOME/.zshrc\n"+
					"export REDTERM=1\n"+
					"export REDTERM_SESSION=%s\n"+
					"PS1=\"%%F{208}[RT:%s]%%f $PS1\"\n",
				sessionID, sessionID,
			)
			zshrc := filepath.Join(zdotdir, ".zshrc")
			if err := os.WriteFile(zshrc, []byte(rcContent), 0600); err == nil {
				cmd := exec.Command(shell)
				cmd.Env = append(baseEnv, "ZDOTDIR="+zdotdir)
				return cmd, func() { os.RemoveAll(zdotdir) }
			}
		}
	}

	// Fallback: env vars only, no PS1 modification
	cmd := exec.Command(shell)
	cmd.Env = baseEnv
	return cmd, noop
}

// handleCommand executes a slash command. Returns (exitSession, cmdToExecute).
// exitSession true means the session should terminate.
// cmdToExecute non-empty means inject that command into the PTY and close the command bar.
func handleCommand(input string, provider llm.Provider, buf *redcontext.Buffer, sc *redcontext.SharedContext, eng *config.Engagement) (bool, string) {
	parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
	command := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	ctx := context.Background()
	snapshot := sc.Snapshot(buf)

	switch command {
	case "suggest":
		ui.PrintThinking()
		result, err := intelligence.Suggest(ctx, provider, snapshot, eng)
		if err != nil {
			ui.PrintError(err.Error())
		} else {
			ui.PrintOverlay("suggest", result)
			cmds := intelligence.ParseSuggestCommands(result)
			if len(cmds) > 0 {
				idx := ui.ReadSelection(len(cmds))
				if idx >= 1 {
					return false, cmds[idx-1]
				}
			}
		}

	case "attack":
		ui.PrintThinking()
		result, err := intelligence.Attack(ctx, provider, snapshot, eng)
		if err != nil {
			ui.PrintError(err.Error())
		} else {
			ui.PrintOverlay("attack", result)
		}

	case "sitrep":
		ui.PrintThinking()
		result, err := intelligence.Sitrep(ctx, provider, snapshot, eng)
		if err != nil {
			ui.PrintError(err.Error())
		} else {
			ui.PrintOverlay("sitrep", result)
		}

	case "engage":
		ui.ReadEngagement(eng)

	case "context":
		if arg == "" {
			ui.PrintError("usage: /context <filepath>")
		} else if err := buf.InjectFile(arg); err != nil {
			ui.PrintError(fmt.Sprintf("cannot read %s: %v", arg, err))
		} else {
			ui.PrintOverlay("context", fmt.Sprintf("loaded into context: %s", arg))
		}

	case "prompt":
		if arg == "" {
			ui.PrintError("usage: /prompt <text>")
		} else {
			buf.InjectText(arg)
			ui.PrintOverlay("prompt", "added to context")
		}

	case "clear":
		buf.Clear()
		ui.PrintOverlay("clear", "context buffer cleared")

	case "exit", "quit":
		ui.PrintOverlay("exit", "closing session")
		return true, ""

	case "help":
		ui.PrintOverlay("help",
			"/suggest          — recommend next command\n"+
				"/attack           — brainstorm attack paths\n"+
				"/sitrep           — situational awareness report\n"+
				"/engage           — set/update engagement context\n"+
				"/context <file>   — inject file into context\n"+
				"/prompt <text>    — manually add text to context\n"+
				"/clear            — clear the context buffer\n"+
				"/exit             — close the session\n"+
				"/help             — show this help\n"+
				"\nEsc or empty Enter closes the command bar")

	default:
		ui.PrintError(fmt.Sprintf("unknown command %q — type /help", command))
	}
	return false, ""
}
