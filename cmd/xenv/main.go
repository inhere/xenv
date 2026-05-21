package main

import (
	"github.com/inhere/xenv/internal/cli"
	"github.com/inhere/xenv/internal/xenv/xenvcom"
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
	xenvcom.SetBinName("xenv")
	xenvcom.SetBinCommand("xenv")

	cli.NewApp().Run(nil)
}
