package cmd

import (
	"fmt"
	"slices"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/tez-capital/tezbake/apps"
	"github.com/tez-capital/tezbake/apps/base"
)

type AppInitialSelectionType string

const (
	InstalledApps AppInitialSelectionType = "installed"
	AllApps       AppInitialSelectionType = "all"
)

type AppFallbackSelectionType string

const (
	NoFallback   AppFallbackSelectionType = "none"
	ImplicitApps AppFallbackSelectionType = "implicit"
	AllFallback  AppFallbackSelectionType = "all"
)

type AppOptionCheckType string

const (
	NoOptionCheck   AppOptionCheckType = "none"
	InfoOptionCheck AppOptionCheckType = "info"
)

type AppSelectionCriteria struct {
	InitialSelection  AppInitialSelectionType
	FallbackSelection AppFallbackSelectionType
	OptionCheckType   AppOptionCheckType
}

// GetAppsBySelectionCriteria filters a collection of apps based on the specified selection criteria,
// including the initial selection set, option check types, and fallback selection mechanisms.
//
// Parameters:
//   - cmd: A *cobra.Command instance from which flag values are retrieved. This is used to determine which apps
//     have been explicitly selected by the user through command-line flags.
//   - criteria: A AppSelectionCriteria struct that defines the criteria for selecting apps, including
//     fields for initial selection (InstalledApps or AllApps), fallback selection (NoFallback,
//     ImplicitApps, or AllFallback), and option check type (NoOptionCheck or InfoOptionCheck).
//
// Returns:
// A slice of base.BakeBuddyApp instances that meet the specified selection criteria. If no apps match the initial
// selection and option check criteria, the fallback selection is used to determine the final set of apps.
//
// Example:
//
//  1. Initial selection set to InstalledApps, with no specific options checked, and no fallback:
//     criteria := AppSelectionCriteria{InitialSelection: InstalledApps, OptionCheckType: NoOptionCheck, FallbackSelection: NoFallback}
//     selectedApps := FilterAppsBySelectionCriteria(cmd, criteria)
//     // This will return all installed apps if the user has not specified any apps via flags.
//
//  2. Initial selection set to AllApps, checking for InfoOptionCheck, with a fallback to ImplicitApps:
//     criteria := AppSelectionCriteria{InitialSelection: AllApps, OptionCheckType: InfoOptionCheck, FallbackSelection: ImplicitApps}
//     selectedApps := FilterAppsBySelectionCriteria(cmd, criteria)
//     // This will return apps based on user flags, apps with 'info' options if flagged, or implicit apps as a fallback.
func GetAppsBySelectionCriteria(cmd *cobra.Command, criteria AppSelectionCriteria) []base.BakeBuddyApp {
	var initialApps []base.BakeBuddyApp
	switch criteria.InitialSelection {
	case InstalledApps:
		initialApps = apps.GetInstalledApps()
	case AllApps:
		initialApps = apps.All
	}

	selectedApps := make([]base.BakeBuddyApp, 0, len(initialApps))
	anyAppSelected := false
	for _, app := range initialApps {
		if checked, _ := cmd.Flags().GetBool(app.GetId()); checked {
			selectedApps = append(selectedApps, app)
			anyAppSelected = true
			continue
		}

		switch criteria.OptionCheckType {
		case InfoOptionCheck:
			infoOptions := app.GetAvailableInfoCollectionOptions()
			checked := false
			for _, option := range infoOptions {
				if option.Type == "bool" {
					if checked, _ := cmd.Flags().GetBool(fmt.Sprintf("%s-%s", app.GetId(), option.Name)); checked {
						selectedApps = append(selectedApps, app)
						checked = true
						break
					}
				}
			}
			if checked {
				continue
			}
		default:
			continue
		}
	}
	if !anyAppSelected && criteria.InitialSelection == InstalledApps {
		// If no apps were selected and the initial selection was installed apps, we should return all installed apps.
		selectedApps = initialApps
	}

	if len(selectedApps) == 0 {
		var fallbackApps []base.BakeBuddyApp
		switch criteria.FallbackSelection {
		case NoFallback:
			return []base.BakeBuddyApp{}
		case ImplicitApps:
			fallbackApps = apps.Implicit
		case AllFallback:
			fallbackApps = apps.All
		}

		selectedApps = lo.Filter(initialApps, func(app base.BakeBuddyApp, _ int) bool {
			return slices.Contains(fallbackApps, app)
		})
	}
	return selectedApps
}

type BoolStringCombinedFlag struct {
	valueB bool
	valueS string
}

// String is required for pflag.Value interface.
func (i *BoolStringCombinedFlag) String() string {
	return i.valueS
}

// Set is required for pflag.Value interface.
func (i *BoolStringCombinedFlag) Set(value string) error {
	i.valueS = value
	i.valueB = value != "" && value != "false"
	return nil
}

func (i *BoolStringCombinedFlag) IsBoolFlag() bool { return true }

func (i *BoolStringCombinedFlag) Type() string {
	return "[string]"
}

func (i *BoolStringCombinedFlag) IsTrue() bool {
	return i.valueB
}

func (i *BoolStringCombinedFlag) HasValue() bool {
	return i.valueS != "" && i.valueS != "false" && i.valueS != "true"
}

func (i *BoolStringCombinedFlag) Value() string {
	return i.valueS
}

func addCombinedFlag(cmd *cobra.Command, name string, value string, usage string) *BoolStringCombinedFlag {
	flag := &BoolStringCombinedFlag{valueS: value}
	f := cmd.Flags().VarPF(flag, name, "", usage)
	f.NoOptDefVal = "true"
	return flag
}
