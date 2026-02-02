package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/cli"
	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/logging"
	"github.com/tez-capital/tezbake/system"
	"github.com/tez-capital/tezbake/util"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Snapshot configuration
const snapshotBaseURL = "https://snapshots.tzinit.org"

// Network and snapshot configuration
type network struct {
	name  string
	modes []string // available snapshot modes
}

var availableNetworks = []network{
	{name: "mainnet", modes: []string{"rolling", "full"}},
	{name: "ghostnet", modes: []string{"rolling"}},
	{name: "shadownet", modes: []string{"rolling", "full"}},
}

// Quick option represents a preset bootstrap configuration
type quickOption struct {
	label       string
	description string
	network     string
	mode        string
	noCheck     bool
	isAdvanced  bool
}

var quickOptions = []quickOption{
	{
		label:       "Mainnet Rolling (Secure)",
		description: "Recommended - Downloads mainnet rolling snapshot with integrity verification",
		network:     "mainnet",
		mode:        "rolling",
		noCheck:     false,
	},
	{
		label:       "Mainnet Rolling (Fast)",
		description: "Skip integrity verification for faster bootstrap",
		network:     "mainnet",
		mode:        "rolling",
		noCheck:     true,
	},
	{
		label:       "Advanced",
		description: "Choose network, snapshot type, and verification options",
		isAdvanced:  true,
	},
}

// Model states
type bootstrapState int

const (
	stateQuickSelect bootstrapState = iota
	stateNetworkSelect
	stateModeSelect
	stateCheckSelect
	stateKeepSnapshotSelect
	stateDone
	stateCanceled
)

// Bootstrap TUI model
type bootstrapModel struct {
	state        bootstrapState
	cursor       int
	selectedNet  network
	selectedMode string
	noCheck      bool
	keepSnapshot bool
	nodePath     string // path to the node being bootstrapped (shown in title if non-default)

	// For display
	quickOptions        []quickOption
	networks            []network
	modes               []string
	checkOptions        []string
	keepSnapshotOptions []string
}

func newBootstrapModel(nodePath string) bootstrapModel {
	return bootstrapModel{
		state:               stateQuickSelect,
		quickOptions:        quickOptions,
		networks:            availableNetworks,
		checkOptions:        []string{"Verify integrity (recommended)", "Skip verification (faster)"},
		keepSnapshotOptions: []string{"Delete snapshot after import (default)", "Keep snapshot on disk"},
		nodePath:            nodePath,
	}
}

func (m bootstrapModel) Init() tea.Cmd {
	return nil
}

func (m bootstrapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.state = stateCanceled
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			maxItems := m.getMaxItems()
			if m.cursor < maxItems-1 {
				m.cursor++
			}

		case "enter", " ":
			return m.handleSelection()
		}
	}
	return m, nil
}

func (m bootstrapModel) getMaxItems() int {
	switch m.state {
	case stateQuickSelect:
		return len(m.quickOptions)
	case stateNetworkSelect:
		return len(m.networks)
	case stateModeSelect:
		return len(m.modes)
	case stateCheckSelect:
		return len(m.checkOptions)
	case stateKeepSnapshotSelect:
		return len(m.keepSnapshotOptions)
	}
	return 0
}

func (m bootstrapModel) handleSelection() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateQuickSelect:
		selected := m.quickOptions[m.cursor]
		if selected.isAdvanced {
			m.state = stateNetworkSelect
			m.cursor = 0
		} else {
			// Quick option selected - find the network
			for _, net := range m.networks {
				if net.name == selected.network {
					m.selectedNet = net
					break
				}
			}
			m.selectedMode = selected.mode
			m.noCheck = selected.noCheck
			m.state = stateDone
			return m, tea.Quit
		}

	case stateNetworkSelect:
		m.selectedNet = m.networks[m.cursor]
		m.modes = m.selectedNet.modes
		if len(m.modes) == 1 {
			// Only one mode available, auto-select it
			m.selectedMode = m.modes[0]
			m.state = stateCheckSelect
		} else {
			m.state = stateModeSelect
		}
		m.cursor = 0

	case stateModeSelect:
		m.selectedMode = m.modes[m.cursor]
		m.state = stateCheckSelect
		m.cursor = 0

	case stateCheckSelect:
		m.noCheck = m.cursor == 1 // Index 1 is "Skip verification"
		m.state = stateKeepSnapshotSelect
		m.cursor = 0

	case stateKeepSnapshotSelect:
		m.keepSnapshot = m.cursor == 1 // Index 1 is "Keep snapshot"
		m.state = stateDone
		return m, tea.Quit
	}
	return m, nil
}

func (m bootstrapModel) View() string {
	var s strings.Builder

	switch m.state {
	case stateQuickSelect:
		title := "🥯 TezBake Node Bootstrap"
		if m.nodePath != "" {
			title = fmt.Sprintf("🥯 TezBake Node Bootstrap [%s]", m.nodePath)
		}
		s.WriteString(constants.StyleTitle.Render(title))
		s.WriteString("\n\n")
		s.WriteString("Select bootstrap option:\n\n")

		for i, opt := range m.quickOptions {
			cursor := "  "
			style := constants.StyleNormal
			if m.cursor == i {
				cursor = "▸ "
				style = constants.StyleSelected
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(opt.label)))
			if m.cursor == i {
				s.WriteString(fmt.Sprintf("    %s\n", constants.StyleDim.Render(opt.description)))
			}
		}

	case stateNetworkSelect:
		s.WriteString(constants.StyleTitle.Render("🌐 Select Network"))
		s.WriteString("\n\n")

		for i, net := range m.networks {
			cursor := "  "
			style := constants.StyleNormal
			if m.cursor == i {
				cursor = "▸ "
				style = constants.StyleSelected
			}
			modeInfo := constants.StyleDim.Render(fmt.Sprintf(" (%s)", strings.Join(net.modes, ", ")))
			s.WriteString(fmt.Sprintf("%s%s%s\n", cursor, style.Render(net.name), modeInfo))
		}

	case stateModeSelect:
		s.WriteString(constants.StyleTitle.Render("📦 Select Snapshot Type"))
		s.WriteString("\n\n")
		s.WriteString(constants.StyleDim.Render(fmt.Sprintf("Network: %s\n\n", m.selectedNet.name)))

		for i, mode := range m.modes {
			cursor := "  "
			style := constants.StyleNormal
			if m.cursor == i {
				cursor = "▸ "
				style = constants.StyleSelected
			}
			desc := ""
			if mode == "rolling" {
				desc = constants.StyleDim.Render(" - smaller, recent history only")
			} else if mode == "full" {
				desc = constants.StyleDim.Render(" - complete block history")
			}
			s.WriteString(fmt.Sprintf("%s%s%s\n", cursor, style.Render(mode), desc))
		}

	case stateCheckSelect:
		s.WriteString(constants.StyleTitle.Render("🔒 Verification Option"))
		s.WriteString("\n\n")
		s.WriteString(constants.StyleDim.Render(fmt.Sprintf("Network: %s, Type: %s\n\n", m.selectedNet.name, m.selectedMode)))

		for i, opt := range m.checkOptions {
			cursor := "  "
			style := constants.StyleNormal
			if m.cursor == i {
				cursor = "▸ "
				style = constants.StyleSelected
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(opt)))
		}

	case stateKeepSnapshotSelect:
		s.WriteString(constants.StyleTitle.Render("💾 Keep Snapshot?"))
		s.WriteString("\n\n")
		s.WriteString(constants.StyleDim.Render(fmt.Sprintf("Network: %s, Type: %s, Verify: %s\n\n", m.selectedNet.name, m.selectedMode, map[bool]string{true: "no", false: "yes"}[m.noCheck])))

		for i, opt := range m.keepSnapshotOptions {
			cursor := "  "
			style := constants.StyleNormal
			if m.cursor == i {
				cursor = "▸ "
				style = constants.StyleSelected
			}
			s.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(opt)))
		}
	}

	s.WriteString(constants.StyleHelp.Render("\n↑/↓ navigate • enter select • q quit"))

	return s.String()
}

// Build snapshot URL from selections
func buildSnapshotURL(network, mode string) string {
	return fmt.Sprintf("%s/%s/%s", snapshotBaseURL, network, mode)
}

// snapshotSelection holds the result from the interactive selector
type snapshotSelection struct {
	url          string
	noCheck      bool
	keepSnapshot bool
	canceled     bool
}

// Run the interactive snapshot selector
func runSnapshotSelector(nodePath string) snapshotSelection {
	model := newBootstrapModel(nodePath)
	result, err := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		logging.Error("Error running snapshot selector:", "error", err)
		return snapshotSelection{canceled: true}
	}

	finalModel, ok := result.(bootstrapModel)
	if !ok {
		logging.Error("Unexpected model type from snapshot selector")
		return snapshotSelection{canceled: true}
	}

	if finalModel.state == stateCanceled {
		return snapshotSelection{canceled: true}
	}

	url := buildSnapshotURL(finalModel.selectedNet.name, finalModel.selectedMode)
	return snapshotSelection{
		url:          url,
		noCheck:      finalModel.noCheck,
		keepSnapshot: finalModel.keepSnapshot,
	}
}

var bootstrapNodeCmd = &cobra.Command{
	Use:   "bootstrap-node [--no-check] [--keep-snapshot] [<url or path>] [<block hash>]",
	Short: "Bootstraps Bake Buddy's Tezos node.",
	Long: `Downloads bootstrap snapshot and imports it into node database.

The source can be either a URL or a local file path to a snapshot.
Optionally, a block hash can be provided for verification.

If no source is provided and running in a TTY, an interactive selector will be shown
to help you choose the appropriate snapshot for your needs.`,
	Args:      cobra.MaximumNArgs(2),
	ValidArgs: []string{"url", "block hash"},
	Run: func(cmd *cobra.Command, args []string) {
		disableSnapshotCheck, _ := cmd.Flags().GetBool("no-check")
		keepSnapshot, _ := cmd.Flags().GetBool("keep-snapshot")

		var snapshotSource string
		var blockHash string

		// Determine if we're bootstrapping a non-default instance
		var nodePath string
		if cli.BBdir != constants.DefaultBBDirectory {
			nodePath = apps.Node.GetPath()
		}

		// If no arguments provided and we're in a TTY, show interactive selector
		if len(args) == 0 && system.IsTty() {
			selection := runSnapshotSelector(nodePath)
			if selection.canceled {
				logging.Info("Bootstrap canceled.")
				os.Exit(0)
			}
			snapshotSource = selection.url
			if selection.noCheck {
				disableSnapshotCheck = true
			}
			if selection.keepSnapshot {
				keepSnapshot = true
			}
		} else if len(args) > 0 {
			snapshotSource = args[0]
			if len(args) > 1 {
				blockHash = args[1]
			}
		} else {
			logging.Error("No snapshot URL or path provided. Use --help for usage information.")
			os.Exit(1)
		}

		if nodePath != "" {
			logging.Info("Bootstrapping node at:", "node_path", nodePath)
		}
		logging.Info("Bootstrapping from:", "snapshot_source", snapshotSource)
		if blockHash != "" {
			logging.Info("Block hash:", "hash", blockHash)
		}
		if disableSnapshotCheck {
			logging.Warn("Snapshot integrity verification disabled")
		}
		if keepSnapshot {
			logging.Info("Snapshot will be kept on disk after import")
		}

		// Check if node was running and stop it before bootstrap
		wasRunning, _ := apps.Node.IsAnyServiceStatus("running")
		if wasRunning {
			logging.Info("Stopping node for bootstrap...")
			exitCode, err := apps.Node.Stop()
			util.AssertEE(err, "Failed to stop node before bootstrap", exitCode)
		}

		bootstrapArgs := []string{"bootstrap", snapshotSource}
		if blockHash != "" {
			bootstrapArgs = append(bootstrapArgs, blockHash)
		}
		if disableSnapshotCheck {
			bootstrapArgs = append(bootstrapArgs, "--no-check")
		}
		if keepSnapshot {
			bootstrapArgs = append(bootstrapArgs, "--keep-snapshot")
		}

		exitCode, err := apps.Node.Execute(bootstrapArgs...)
		util.AssertEE(err, "Failed to bootstrap tezos node", exitCode)

		logging.Info("Upgrading storage...")
		exitCode, err = apps.Node.UpgradeStorage()
		util.AssertEE(err, "Failed to upgrade tezos storage", exitCode)

		// Restart node if it was running before bootstrap
		if wasRunning {
			logging.Info("Restarting node...")
			exitCode, err = apps.Node.Start()
			util.AssertEE(err, "Failed to restart node after bootstrap", exitCode)
		}

		os.Exit(exitCode)
	},
}

func init() {
	bootstrapNodeCmd.Flags().Bool("no-check", false, "Bootstrap node without verifying snapshot integrity")
	bootstrapNodeCmd.Flags().Bool("keep-snapshot", false, "Keep the snapshot file on disk after import")
	RootCmd.AddCommand(bootstrapNodeCmd)
}
