package manager

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/gookit/ext/tomlkit"
	"github.com/gookit/goutil/fsutil"
	"github.com/inhere/xenv/internal/xenv/models"
)

// StateTomlUpdater writes an ActivityState back to its TOML file.
//
// Only what actually changed is applied: tables whose content is unchanged keep
// their original text (comments, key order and formatting included), inside a
// changed table untouched keys keep their lines, and tables the model does not
// know about — a hand-written section in a project .xenv.toml — are left as they
// are. Empty sections are rendering defaults and are not added to a file that
// never had them.
//
// The merging is provided by github.com/gookit/ext/tomlkit.
type StateTomlUpdater struct{}

// NewTomlUpdater creates a new StateTomlUpdater
func NewTomlUpdater() *StateTomlUpdater { return &StateTomlUpdater{} }

// Update applies state to its state file.
func (u *StateTomlUpdater) Update(state *models.ActivityState) error {
	rendered, err := renderState(state)
	if err != nil {
		return err
	}
	defaults, err := renderState(models.NewActivityState(""))
	if err != nil {
		return err
	}

	return tomlkit.MergeFile(state.File, rendered, tomlkit.Options{
		Decode:          decodeStateToml,
		Defaults:        defaults,
		KeepExtraTables: true,
	})
}

// WriteNewState writes a fresh state file.
func (u *StateTomlUpdater) WriteNewState(state *models.ActivityState) error {
	contents, err := renderState(state)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(state.File, []byte(contents), 0o644)
}

func renderState(state *models.ActivityState) (string, error) {
	contents, err := toml.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}
	return string(contents), nil
}

// decodeStateToml parses a state document so the merger can tell which parts of
// it changed.
func decodeStateToml(text string) (map[string]any, error) {
	data := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &data); err != nil {
		return nil, err
	}
	return data, nil
}
