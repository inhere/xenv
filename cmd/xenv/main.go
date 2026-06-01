package main

import (
	"os"

	"github.com/inhere/xenv/internal/cli"
)

// Build-time variables injected via -ldflags
var (
	Version   = "dev"
	GitHash   = "unknown"
	BuildTime = "unknown"
)

// main xenv 程序入口
//
// Dev run:
//
//	go run ./cmd/xenv
//	go run ./cmd/xenv <CMD>
//
// Debug run:
//
//		KITE_VERBOSE=debug go run ./cmd/xenv <CMD>
//	 // Windows PowerShell
//		$env:KITE_VERBOSE="debug" go run ./cmd/xenv <CMD>
func main() {
	cli.SetBuildInfo(Version, GitHash, BuildTime)

	os.Exit(cli.NewApp().Run(nil))
}
