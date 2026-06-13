package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
)

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
