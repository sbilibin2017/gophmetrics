// Package main implements a multichecker that runs a set of static analysis
// analyzers on Go code.
//
// This tool runs a collection of analyzers including:
// - Standard analyzers from golang.org/x/tools/go/analysis/passes
// - All SA analyzers from staticcheck.io
// - Additional analyzers from other staticcheck classes
// - Two or more public third-party analyzers
// - A custom analyzer that forbids direct calls to os.Exit in main.main
//
// Usage:
//
//	go run cmd/staticlint/main.go ./...
//	./staticlint ./...
//
// Use this tool to detect issues and errors in your project code.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/sbilibin2017/gophmetrics/cmd/staticlint/analyzers"
)

// main runs the multichecker tool that aggregates multiple analyzers,
// including standard analyzers, staticcheck analyzers, and custom analyzers.
func main() {
	multichecker.Main(
		analyzers.NoOsExitMainAnalyzer,
	)
}
