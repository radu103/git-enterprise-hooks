package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

func CanPrompt() bool {
	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return true
	}
	termIn, termOut := openTerminalIO()
	if termIn == nil {
		return false
	}
	_ = termIn.Close()
	_ = termOut.Close()
	return true
}

func openTerminalIO() (in *os.File, out *os.File) {
	if runtime.GOOS == "windows" {
		inFile, inErr := os.Open("CONIN$")
		if inErr != nil {
			return nil, nil
		}
		outFile, outErr := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
		if outErr != nil {
			_ = inFile.Close()
			return nil, nil
		}
		return inFile, outFile
	}

	inFile, inErr := os.Open("/dev/tty")
	if inErr != nil {
		return nil, nil
	}
	outFile, outErr := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if outErr != nil {
		_ = inFile.Close()
		return nil, nil
	}
	return inFile, outFile
}

func Ask(question, defaultValue string) (string, error) {
	in := os.Stdin
	out := os.Stdout
	termIn, termOut := openTerminalIO()
	if termIn != nil {
		defer termIn.Close()
		defer termOut.Close()
		in = termIn
		out = termOut
	}

	reader := bufio.NewReader(in)
	if strings.TrimSpace(defaultValue) != "" {
		_, _ = fmt.Fprintf(out, "%s [%s]: ", question, defaultValue)
	} else {
		_, _ = fmt.Fprintf(out, "%s: ", question)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			v := strings.TrimSpace(line)
			if v == "" {
				return defaultValue, nil
			}
			return v, nil
		}
		return "", err
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return defaultValue, nil
	}
	return v, nil
}

func AskPassword(question string) (string, error) {
	in := os.Stdin
	out := os.Stdout
	termIn, termOut := openTerminalIO()
	if termIn != nil {
		defer termIn.Close()
		defer termOut.Close()
		in = termIn
		out = termOut
	}

	_, _ = fmt.Fprintf(out, "%s: ", question)
	b, err := term.ReadPassword(int(in.Fd()))
	_, _ = fmt.Fprintln(out)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func Confirm(question string) (bool, error) {
	v, err := Ask(question+" (yes/no)", "no")
	if err != nil {
		return false, err
	}
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "yes" || v == "y", nil
}
