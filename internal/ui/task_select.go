package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/radu103/git-enterprise-hooks/internal/domain"
	"github.com/radu103/git-enterprise-hooks/internal/errs"
	"golang.org/x/term"
)

type taskItem struct {
	task domain.Task
}

func (i taskItem) Title() string { return fmt.Sprintf("%s - %s", i.task.Key, i.task.Title) }
func (i taskItem) Description() string {
	return fmt.Sprintf("Epic: %s | Status: %s", i.task.Epic, i.task.Status)
}
func (i taskItem) FilterValue() string { return i.task.Key + " " + i.task.Title }

type selectModel struct {
	list     list.Model
	selected *domain.Task
	quit     bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			it, ok := m.list.SelectedItem().(taskItem)
			if ok {
				v := it.task
				m.selected = &v
				m.quit = true
				return m, tea.Quit
			}
		case "q", "ctrl+c":
			m.quit = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Select One Task")
	return title + "\n\n" + m.list.View()
}

func SelectTask(tasks []domain.Task) (*domain.Task, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	if len(tasks) == 1 {
		selected := tasks[0]
		fmt.Printf("Auto-selected task: %s - %s\n", selected.Key, selected.Title)
		return &selected, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, fmt.Errorf(errs.TaskSelectionRequiresInteractive)
	}

	items := make([]list.Item, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, taskItem{task: t})
	}
	l := list.New(items, list.NewDefaultDelegate(), 80, 18)
	l.Title = "Tasks"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	m := selectModel{list: l}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := final.(selectModel)
	if result.selected == nil {
		return nil, nil
	}
	return result.selected, nil
}
