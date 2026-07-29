package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func Ask(question, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	if strings.TrimSpace(defaultValue) != "" {
		fmt.Printf("%s [%s]: ", question, defaultValue)
	} else {
		fmt.Printf("%s: ", question)
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
	fmt.Printf("%s: ", question)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
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
