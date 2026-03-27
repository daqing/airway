package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type replInput struct {
	in      *os.File
	out     io.Writer
	reader  *bufio.Reader
	history []string
}

func newREPLInput(in *os.File, out io.Writer) *replInput {
	return &replInput{
		in:     in,
		out:    out,
		reader: bufio.NewReader(in),
	}
}

func (r *replInput) ReadLine(prompt string) (string, error) {
	if !isTerminal(r.in) {
		fmt.Fprint(r.out, prompt)
		line, err := r.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && line != "" {
				return line, nil
			}

			return "", err
		}

		return trimTrailingNewline(line), nil
	}

	return r.readTerminalLine(prompt)
}

func (r *replInput) readTerminalLine(prompt string) (string, error) {
	state, err := makeRaw(r.in.Fd())
	if err != nil {
		return "", err
	}

	defer restoreTerminal(r.in.Fd(), state)

	if _, err := fmt.Fprint(r.out, prompt); err != nil {
		return "", err
	}

	buffer := []rune{}
	cursor := 0
	historyIndex := len(r.history)
	draft := ""

	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			if err == io.EOF && len(buffer) == 0 {
				return "", io.EOF
			}

			return "", err
		}

		switch b {
		case '\r', '\n':
			line := string(buffer)
			if _, err := fmt.Fprint(r.out, "\r\n"); err != nil {
				return "", err
			}

			r.appendHistory(line)
			return line, nil
		case 4:
			if len(buffer) == 0 {
				return "", io.EOF
			}
		case 127, '\b':
			if cursor == 0 || len(buffer) == 0 {
				continue
			}

			buffer = append(buffer[:cursor-1], buffer[cursor:]...)
			cursor--
			draft = string(buffer)
			if err := r.renderLine(prompt, draft, cursor); err != nil {
				return "", err
			}
			historyIndex = len(r.history)
		case 27:
			sequence, seqErr := r.readEscapeSequence()
			if seqErr != nil {
				return "", seqErr
			}

			switch sequence {
			case "[A":
				if len(r.history) == 0 || historyIndex == 0 {
					continue
				}
				if historyIndex == len(r.history) {
					draft = string(buffer)
				}
				historyIndex--
				buffer = []rune(r.history[historyIndex])
				cursor = len(buffer)
			case "[B":
				if len(r.history) == 0 || historyIndex >= len(r.history) {
					continue
				}
				historyIndex++
				if historyIndex == len(r.history) {
					buffer = []rune(draft)
				} else {
					buffer = []rune(r.history[historyIndex])
				}
				cursor = len(buffer)
			case "[C":
				if cursor < len(buffer) {
					cursor++
					if err := r.renderLine(prompt, string(buffer), cursor); err != nil {
						return "", err
					}
				}
				continue
			case "[D":
				if cursor > 0 {
					cursor--
					if err := r.renderLine(prompt, string(buffer), cursor); err != nil {
						return "", err
					}
				}
				continue
			default:
				continue
			}

			if err := r.renderLine(prompt, string(buffer), cursor); err != nil {
				return "", err
			}
		default:
			if b < 32 {
				continue
			}

			buffer = append(buffer[:cursor], append([]rune{rune(b)}, buffer[cursor:]...)...)
			cursor++
			draft = string(buffer)
			if err := r.renderLine(prompt, draft, cursor); err != nil {
				return "", err
			}
			historyIndex = len(r.history)
		}
	}
}

func (r *replInput) readEscapeSequence() (string, error) {
	first, err := r.reader.ReadByte()
	if err != nil {
		return "", err
	}

	if first != '[' {
		return string([]byte{first}), nil
	}

	second, err := r.reader.ReadByte()
	if err != nil {
		return "", err
	}

	return string([]byte{first, second}), nil
}

func (r *replInput) renderLine(prompt string, line string, cursor int) error {
	if _, err := fmt.Fprintf(r.out, "\r\033[2K%s%s", prompt, line); err != nil {
		return err
	}

	tail := len([]rune(line)) - cursor
	if tail > 0 {
		_, err := fmt.Fprintf(r.out, "\033[%dD", tail)
		return err
	}

	return nil
}

func (r *replInput) appendHistory(line string) {
	if line == "" {
		return
	}

	if len(r.history) > 0 && r.history[len(r.history)-1] == line {
		return
	}

	r.history = append(r.history, line)
}

func isTerminal(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), ioctlReadTermios)
	return err == nil
}

func makeRaw(fd uintptr) (*unix.Termios, error) {
	state, err := unix.IoctlGetTermios(int(fd), ioctlReadTermios)
	if err != nil {
		return nil, err
	}

	raw := *state
	raw.Iflag &^= unix.BRKINT | unix.ICRNL | unix.INPCK | unix.ISTRIP | unix.IXON
	raw.Cflag |= unix.CS8
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.IEXTEN
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(int(fd), ioctlWriteTermios, &raw); err != nil {
		return nil, err
	}

	return state, nil
}

func restoreTerminal(fd uintptr, state *unix.Termios) {
	if state == nil {
		return
	}

	_ = unix.IoctlSetTermios(int(fd), ioctlWriteTermios, state)
}

func trimTrailingNewline(line string) string {
	return strings.TrimRight(line, "\r\n")
}
