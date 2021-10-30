package main

import (
	"fmt"
	"os"

	"github.com/docopt/docopt-go"
)

const AppVersion = "0.2"
const description = `
The lipo-app tool creates one universal (multi-architecture) bundle from one or more input bindles.
All of the architectures in each input bundle will be copied into the output bundle.

The lipo-app will only ever write to a single output bundle, and input files are never
modified in place.

`

const usage = `
Usage:
  lipo-app [options] <input_bundle> <input_bundle> <output_bundle>
  lipo-app [options] <input_bundle> <output_bundle>
  lipo-app -h | --help
  lipo-app --version

Options:
  -h --help     Show this screen.
  --version     Show version.
  -v --verbose  Prints detailed information about the operations.
`

type cmdArgs struct {
	Input_bundle  []string
	Output_bundle string
	Verbose       bool
}

const (
	RetArgsParseError = 1
	RetCommandError   = 2
	RetExecNotFound   = 3
)

func main() {

	args := cmdArgs{}

	parser := docopt.Parser{}
	opts, _ := parser.ParseArgs(description+usage, os.Args[1:], AppVersion)

	if err := opts.Bind(&args); err != nil {
		parser.HelpHandler(err, usage)
		os.Exit(RetArgsParseError)
	}

	engine, err := newEngine(args.Input_bundle, args.Output_bundle)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(RetArgsParseError)
	}

	engine.verbose = args.Verbose

	err = engine.run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(RetArgsParseError)
	}
}
