package util

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var errPromptCanceled = errors.New("prompt canceled")

func PromptConfirm(message string, defaultValue bool) (bool, error) {
	model := newConfirmModel(message, defaultValue)
	result, err := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return false, err
	}
	finalModel := result.(confirmModel)
	if finalModel.canceled {
		return false, errPromptCanceled
	}
	return parseConfirmValue(finalModel.input.Value(), defaultValue), nil
}

func PromptPassword(message string) (string, error) {
	model := newPasswordModel(message)
	result, err := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return "", err
	}
	finalModel := result.(passwordModel)
	if finalModel.canceled {
		return "", errPromptCanceled
	}
	return finalModel.input.Value(), nil
}

type confirmModel struct {
	prompt       string
	defaultValue bool
	input        textinput.Model
	canceled     bool
}

func newConfirmModel(prompt string, defaultValue bool) confirmModel {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 5
	input.Focus()
	return confirmModel{
		prompt:       prompt,
		defaultValue: defaultValue,
		input:        input,
	}
}

func parseConfirmValue(raw string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return defaultValue
	}
	if value == "1" || value == "true" || strings.HasPrefix(value, "y") {
		return true
	}
	if value == "0" || value == "false" || strings.HasPrefix(value, "n") {
		return false
	}
	return defaultValue
}

func (m confirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m confirmModel) View() string {
	hint := "y/N"
	if m.defaultValue {
		hint = "Y/n"
	}
	return fmt.Sprintf("%s [%s] %s\n", m.prompt, hint, m.input.View())
}

type passwordModel struct {
	prompt   string
	input    textinput.Model
	canceled bool
}

func newPasswordModel(prompt string) passwordModel {
	input := textinput.New()
	input.Prompt = ""
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	input.Focus()
	return passwordModel{
		prompt: prompt,
		input:  input,
	}
}

func (m passwordModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m passwordModel) View() string {
	return fmt.Sprintf("%s %s\n", m.prompt, m.input.View())
}
