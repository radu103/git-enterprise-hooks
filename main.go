package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/radu103/git-enterprise-hooks/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		fmt.Fprintln(os.Stderr, errorStyle.Render("Error:"), err)
		os.Exit(1)
	}
}
