package cli

import (
	"fmt"
	"sort"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv/models"
)

const (
	sourceGlobal    = "Global State"
	sourceSession   = "Session Context"
	sourceDirectory = "Directory State"
)

type stateValue struct {
	Source  string
	Version string
}

type effectiveSDKRow struct {
	Name      string
	Version   string
	Source    string
	Overrides []stateValue
}

type statusOptions struct {
	Layers  bool `flag:"desc=Show Global, Directory, and Session Context layers"`
	Runtime bool `flag:"desc=Show runtime detection details"`
}

func StatusCmd() *gcli.Command {
	var opts statusOptions
	return &gcli.Command{
		Name:    "status",
		Desc:    "Show current xenv status for this shell and directory",
		Aliases: []string{"st"},
		Config: func(c *gcli.Command) {
			c.MustFromStruct(&opts)
		},
		Func: func(c *gcli.Command, args []string) error {
			fmt.Println("[Effective State]")
			fmt.Println("No effective state found")
			return nil
		},
	}
}

func buildEffectiveSDKRows(global, session *models.ActivityState, dirStates []*models.ActivityState) []effectiveSDKRow {
	values := make(map[string][]stateValue)

	collectSDKValues(values, sourceGlobal, global)
	collectSDKValues(values, sourceSession, session)
	for _, dirState := range dirStates {
		collectSDKValues(values, sourceDirectory, dirState)
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]effectiveSDKRow, 0, len(names))
	for _, name := range names {
		stack := values[name]
		effective := stack[len(stack)-1]
		overrides := make([]stateValue, 0, len(stack)-1)
		for i := len(stack) - 2; i >= 0; i-- {
			if stack[i].Version != effective.Version {
				overrides = append(overrides, stack[i])
			}
		}
		rows = append(rows, effectiveSDKRow{
			Name:      name,
			Version:   effective.Version,
			Source:    effective.Source,
			Overrides: overrides,
		})
	}
	return rows
}

func collectSDKValues(values map[string][]stateValue, source string, state *models.ActivityState) {
	if state == nil {
		return
	}
	for name, version := range state.SDKs {
		values[name] = append(values[name], stateValue{Source: source, Version: version})
	}
}
