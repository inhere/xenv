package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv"
)

var SDKCmd = &gcli.Command{
	Name:    "sdk",
	Desc:    "Manage local development SDKs",
	Aliases: []string{"sdks"},
	Subs: []*gcli.Command{
		SDKIndexCmd(),
		SDKListCmd(),
		SDKShowCmd(),
		SDKWhereCmd(),
	},
}

func SDKIndexCmd() *gcli.Command {
	return &gcli.Command{
		Name:    "index",
		Help:    "index",
		Desc:    "Index local installed SDKs to metadata",
		Aliases: []string{"refresh", "scan"},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}
			return sdkSvc.IndexLocalSDKs()
		},
	}
}

func SDKListCmd() *gcli.Command {
	var opts = struct {
		All bool `flag:"shorts=a;desc=List all configured SDKs, including uninstalled ones"`
	}{}

	return &gcli.Command{
		Name:    "list",
		Desc:    "List configured SDKs",
		Aliases: []string{"ls"},
		Config: func(c *gcli.Command) {
			c.MustFromStruct(&opts)
		},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}
			return sdkSvc.ListSDKs(opts.All)
		},
	}
}

func SDKShowCmd() *gcli.Command {
	return &gcli.Command{
		Name: "show",
		Help: "show <name>",
		Desc: "Show information about a specific SDK",
		Config: func(c *gcli.Command) {
			c.AddArg("name", "Name of the SDK to show", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}
			return sdkSvc.ShowSDK(c.Arg("name").String())
		},
	}
}

func SDKWhereCmd() *gcli.Command {
	var opts = struct {
		Bin bool `flag:"desc=Print the SDK bin directory instead of install directory"`
	}{}

	return &gcli.Command{
		Name:    "where",
		Help:    "where [--bin] <name:version>",
		Desc:    "Show the path for a specific SDK version",
		Aliases: []string{"which"},
		Config: func(c *gcli.Command) {
			c.BoolOpt(&opts.Bin, "bin", "b", false, "Print the SDK bin directory")
			c.AddArg("spec", "SDK name and version, e.g. go:1.22", true)
		},
		Func: func(c *gcli.Command, args []string) error {
			sdkSvc, err := xenv.SDKService()
			if err != nil {
				return err
			}

			path, err := sdkSvc.WhereSDK(c.Arg("spec").String(), opts.Bin)
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
}
