package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"fearless/internal/core/collection"
	"fearless/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type envVarFlags map[string]any

func (e envVarFlags) String() string {
	return fmt.Sprintf("%v", map[string]any(e))
}

func (e envVarFlags) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) == 2 {
		e[parts[0]] = parts[1]
	} else {
		e[value] = ""
	}
	return nil
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "fearless-cli version %s\n", version.Version)
		return 0

	case "help", "--help", "-h":
		printUsage(stdout)
		return 0

	case "run":
		return handleRun(rest, stdout, stderr)

	case "export":
		return handleExport(rest, stdout, stderr)

	case "import":
		return handleImport(rest, stdout, stderr)

	default:
		// If first arg is a path or looks like a file/directory, default to run
		if !strings.HasPrefix(cmd, "-") {
			return handleRun(args, stdout, stderr)
		}
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", cmd)
		printUsage(stderr)
		return 1
	}
}

func handleRun(args []string, stdout, stderr io.Writer) int {
	args = reorderArgs(args)
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var env string
	fs.StringVar(&env, "env", "", "Environment name to select")
	fs.StringVar(&env, "e", "", "Environment name to select (shorthand)")

	envVars := make(envVarFlags)
	fs.Var(&envVars, "env-var", "Override environment variable key=value (can be specified multiple times)")

	var recursive bool
	fs.BoolVar(&recursive, "recursive", true, "Recursively run subfolders")
	fs.BoolVar(&recursive, "r", true, "Recursively run subfolders (shorthand)")

	var bail bool
	fs.BoolVar(&bail, "bail", false, "Stop execution immediately on first failure")

	var outputFormat string
	fs.StringVar(&outputFormat, "format", "text", "Output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}

	opts := RunOptions{
		Path:      targetPath,
		Env:       env,
		EnvVars:   envVars,
		Recursive: recursive,
		Bail:      bail,
		Format:    outputFormat,
		Stdout:    stdout,
		Stderr:    stderr,
	}

	summary, err := ExecuteRun(context.Background(), opts)
	if err != nil {
		fmt.Fprintf(stderr, "Run error: %v\n", err)
		return 1
	}

	if summary.Failed > 0 {
		return 1
	}
	return 0
}

func handleExport(args []string, stdout, stderr io.Writer) int {
	args = reorderArgs(args)
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var outputFile string
	fs.StringVar(&outputFile, "output", "", "Output file destination (default: stdout)")
	fs.StringVar(&outputFile, "o", "", "Output file destination (shorthand)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetPath := "."
	if fs.NArg() > 0 {
		targetPath = fs.Arg(0)
	}

	col, err := collection.Load(targetPath)
	if err != nil {
		fmt.Fprintf(stderr, "Export error: %v\n", err)
		return 1
	}

	data, err := json.MarshalIndent(col, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "Export serialize error: %v\n", err)
		return 1
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0o644); err != nil {
			fmt.Fprintf(stderr, "Write output file failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Exported collection %q to %s\n", col.Name, outputFile)
	} else {
		_, _ = stdout.Write(data)
		fmt.Fprintln(stdout)
	}
	return 0
}

func handleImport(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: fearless-cli import <path>")
		return 1
	}
	targetPath := args[0]
	col, err := collection.Load(targetPath)
	if err != nil {
		fmt.Fprintf(stderr, "Import failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Successfully verified and imported collection %q (%d items)\n", col.Name, len(col.Items))
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `fearless-cli - Headless API client & collection runner

Usage:
  fearless-cli <command> [arguments]

Commands:
  run [path]             Run a collection or single request file
  export [path]          Export a collection to JSON
  import [path]          Validate and import a collection
  version                Print the fearless-cli version
  help                   Print this help message

Run Flags:
  -e, --env <name>       Select environment
  --env-var <key=value>  Override environment variable (can repeat)
  -r, --recursive        Recursively run folder items (default true)
  --bail                 Stop execution on first failure
  --format <text|json>   Set output format (default text)
`)
}

func reorderArgs(args []string) []string {
	var flags []string
	var pos []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && (arg == "-e" || arg == "--env" || arg == "--env-var" || arg == "--format" || arg == "-o" || arg == "--output") {
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			pos = append(pos, arg)
		}
	}
	return append(flags, pos...)
}
