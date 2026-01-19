package util

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/tez-capital/tezbake/constants"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var ErrPromptCanceled = errors.New("prompt canceled")

func Confirm(message string, defaultValue bool, failureMsg ...string) bool {
	return ConfirmWithCancelValue(message, defaultValue, false, failureMsg...)
}

func ConfirmWithCancelValue(message string, defaultValue bool, cancelValue bool, failureMsg ...string) bool {
	response, err := promptConfirm(message, defaultValue)
	if err != nil {
		if errors.Is(err, ErrPromptCanceled) {
			return cancelValue
		}
		AssertEE(err, confirmFailureMessage(message, failureMsg), constants.ExitInternalError)
	}
	return response
}

func ConfirmOrExit(message string, defaultValue bool, failureMsg ...string) {
	if !Confirm(message, defaultValue, failureMsg...) {
		os.Exit(constants.ExitOperationCanceled)
	}
}

func RequirePasswordE(message string, errMsg string, errExitCode int) string {
	password, err := promptPassword(message)
	if err != nil {
		if errors.Is(err, ErrPromptCanceled) {
			os.Exit(constants.ExitOperationCanceled)
		}
		AssertEE(err, errMsg, errExitCode)
	}
	return password
}

func PromptPasswordE(message string) (string, error) {
	password, err := promptPassword(message)
	if err != nil {
		if errors.Is(err, ErrPromptCanceled) {
			os.Exit(constants.ExitOperationCanceled)
		}
		return "", err
	}
	return password, nil
}

func promptConfirm(message string, defaultValue bool) (bool, error) {
	model := newConfirmModel(message, defaultValue)
	result, err := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return false, err
	}
	finalModel, ok := result.(confirmModel)
	if !ok {
		return false, fmt.Errorf("unexpected confirm model type %T", result)
	}
	if finalModel.canceled {
		return false, ErrPromptCanceled
	}
	return parseConfirmValue(finalModel.input.Value(), defaultValue), nil
}

func promptPassword(message string) (string, error) {
	model := newPasswordModel(message)
	result, err := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return "", err
	}
	finalModel, ok := result.(passwordModel)
	if !ok {
		return "", fmt.Errorf("unexpected password model type %T", result)
	}
	if finalModel.canceled {
		return "", ErrPromptCanceled
	}
	return finalModel.input.Value(), nil
}

func confirmFailureMessage(message string, failureMsg []string) string {
	if len(failureMsg) > 0 && failureMsg[0] != "" {
		return failureMsg[0]
	}
	return "Failed to confirm: " + message
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
	switch value {
	case "y", "yes":
		return true
	case "n", "no":
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
