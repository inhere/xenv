package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gookit/gcli/v3"
	"github.com/gookit/goutil/x/ccolor"
	"github.com/inhere/xenv/internal/xenv"
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
			return handleStatus(opts)
		},
	}
}

func handleStatus(opts statusOptions) error {
	if err := xenv.InitState(); err != nil {
		return fmt.Errorf("failed to load activity state: %w", err)
	}

	global := xenv.State().Global()
	session := xenv.State().Session()
	dirStates := xenv.State().DirStates()

	printStatusSection("Effective State", formatEffectiveSDKRows(buildEffectiveSDKRows(global, session, dirStates)))
	if opts.Runtime {
		fmt.Println()
		printStatusSection("Runtime State", []string{"Runtime detection is not implemented yet"})
	}
	if opts.Layers {
		overridden := buildOverriddenSDKSources(global, session, dirStates)
		fmt.Println()
		printStatusSection("Global State", formatLayerLines("Global Defaults:", global, "No global state found", overridden[global]))
		for _, dirState := range dirStates {
			fmt.Println()
			printStatusSection("Directory State", formatLayerLines("Directory SDKs:", dirState, "No directory state found", overridden[dirState]))
		}
		fmt.Println()
		printStatusSection("Session Context", formatLayerLines("Session Defaults:", session, "No session context found", overridden[session]))
	}
	return nil
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

func formatEffectiveSDKRows(rows []effectiveSDKRow) []string {
	if len(rows) == 0 {
		return []string{"No effective state found"}
	}

	lines := []string{"SDKs:"}
	var overrideLines []string
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("%12s => %s  (%s)", row.Name, row.Version, row.Source))
		if len(row.Overrides) == 0 {
			continue
		}
		parts := make([]string, 0, len(row.Overrides))
		for _, item := range row.Overrides {
			parts = append(parts, fmt.Sprintf("%s %s", item.Source, item.Version))
		}
		overrideLines = append(overrideLines, fmt.Sprintf("%12s: %s", row.Name, strings.Join(parts, ", ")))
	}

	if len(overrideLines) > 0 {
		lines = append(lines, "", "Overrides:")
		lines = append(lines, overrideLines...)
	}
	return lines
}

func formatLayerLines(title string, state *models.ActivityState, emptyMessage string, overridden map[string]string) []string {
	if state == nil || state.IsEmpty() {
		return []string{emptyMessage}
	}
	lines := []string{" - from: " + state.File}
	lines = append(lines, formatStateSDKLines(title, state, overridden)...)
	lines = append(lines, formatEnvPathToolLines(state)...)
	return lines
}

func formatStateSDKLines(title string, state *models.ActivityState, overridden map[string]string) []string {
	if state == nil || len(state.SDKs) == 0 {
		return nil
	}

	lines := []string{title}
	names := make([]string, 0, len(state.SDKs))
	for name := range state.SDKs {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		suffix := ""
		if by := overridden[name]; by != "" {
			suffix = fmt.Sprintf("  (overridden by %s)", by)
		}
		lines = append(lines, fmt.Sprintf("  <green>%10s</> => %s%s", name, state.SDKs[name], suffix))
	}
	return lines
}

func formatEnvPathToolLines(state *models.ActivityState) []string {
	var lines []string
	if len(state.Envs) > 0 {
		lines = appendStatusSection(lines, "Env Defaults:")
		for name, value := range state.Envs {
			lines = append(lines, fmt.Sprintf("  <green>%s</>=%s", name, value))
		}
	}
	if len(state.Paths) > 0 {
		lines = appendStatusSection(lines, "PATH Defaults:")
		for i, path := range state.Paths {
			lines = append(lines, fmt.Sprintf("  <green>%d</>. %s", i+1, path))
		}
	}
	if len(state.ToolRequirements) > 0 {
		lines = appendStatusSection(lines, "Tool Requirements:")
		for name, requirement := range state.ToolRequirements {
			lines = append(lines, fmt.Sprintf("  <green>%s</> => %s", name, requirement))
		}
	}
	return lines
}

func appendStatusSection(lines []string, title string) []string {
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	return append(lines, title)
}

func buildOverriddenSDKSources(global, session *models.ActivityState, dirStates []*models.ActivityState) map[*models.ActivityState]map[string]string {
	result := make(map[*models.ActivityState]map[string]string)
	rows := buildEffectiveSDKRows(global, session, dirStates)

	for _, row := range rows {
		for _, item := range row.Overrides {
			target := stateBySource(item.Source, row.Name, item.Version, global, session, dirStates)
			if target == nil {
				continue
			}
			if result[target] == nil {
				result[target] = make(map[string]string)
			}
			result[target][row.Name] = row.Source
		}
	}
	return result
}

func stateBySource(source, name, version string, global, session *models.ActivityState, dirStates []*models.ActivityState) *models.ActivityState {
	switch source {
	case sourceGlobal:
		return global
	case sourceSession:
		return session
	case sourceDirectory:
		for _, dirState := range dirStates {
			if dirState.SDKs[name] == version {
				return dirState
			}
		}
	}
	return nil
}

func printStatusSection(name string, lines []string) {
	fmt.Printf("[%s]\n", name)
	fmt.Println(strings.Repeat("-", 69))
	for _, line := range lines {
		if strings.HasPrefix(line, "  <green>") {
			ccolor.Println(line)
			continue
		}
		fmt.Println(line)
	}
}
