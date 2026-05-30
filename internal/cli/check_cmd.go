package cli

import (
	"fmt"

	"github.com/gookit/gcli/v3"
	"github.com/inhere/xenv/internal/xenv"
	"github.com/inhere/xenv/internal/xenv/service"
)

var CheckCmd = &gcli.Command{
	Name: "check",
	Desc: "Check active SDKs and project tool requirements",
	Subs: []*gcli.Command{
		CheckSDKCmd(),
		CheckToolsCmd(),
	},
	Func: func(c *gcli.Command, args []string) error {
		sdkErr := runSDKChecks()
		toolErr := runToolChecks(false)
		if sdkErr != nil {
			return sdkErr
		}
		return toolErr
	},
}

func CheckSDKCmd() *gcli.Command {
	return &gcli.Command{
		Name: "sdk",
		Desc: "Check active SDK availability",
		Func: func(c *gcli.Command, args []string) error {
			return runSDKChecks()
		},
	}
}

func CheckToolsCmd() *gcli.Command {
	return &gcli.Command{
		Name: "tools",
		Desc: "Check project tool requirements from .xenv.toml",
		Func: func(c *gcli.Command, args []string) error {
			return runToolChecks(true)
		},
	}
}

func runSDKChecks() error {
	sdkSvc, err := xenv.SDKService()
	if err != nil {
		return err
	}

	checkSvc := service.NewCheckService(sdkSvc)
	results := checkSvc.CheckSDKs(xenv.State().Merged())
	printCheckResults("SDK", results)
	return firstCheckError(results)
}

func runToolChecks(checkVersion bool) error {
	if err := xenv.InitState(); err != nil {
		return err
	}

	checkSvc := service.NewCheckService(nil)
	results := checkSvc.CheckTools(xenv.State().Merged(), checkVersion)
	printCheckResults("Tool", results)
	return firstCheckError(results)
}

func printCheckResults(kind string, results []service.CheckResult) {
	if len(results) == 0 {
		fmt.Printf("No %s checks to run\n", kind)
		return
	}

	for _, result := range results {
		fmt.Printf("[%s] %s: %s\n", result.Status, result.Name, result.Message)
	}
}

func firstCheckError(results []service.CheckResult) error {
	for _, result := range results {
		if result.Status == service.CheckStatusError {
			return fmt.Errorf("%s check failed: %s", result.Name, result.Message)
		}
	}
	return nil
}
