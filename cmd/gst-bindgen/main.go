package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/codegen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: gst-bindgen <command> [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  generate    Generate Go bindings from GIR files\n")
	fmt.Fprintf(os.Stderr, "  validate    Check that generated files are up to date\n")
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	girDir := fs.String("gir-dir", "gir-files", "Directory containing GIR XML files")
	outputDir := fs.String("output-dir", ".", "Root output directory (containing gst/ subdirectory)")
	overridesDir := fs.String("overrides-dir", "overrides", "Directory containing override YAML files")
	fs.Parse(args)

	cfg := &codegen.Config{
		GIRDir:       *girDir,
		OutputDir:    *outputDir,
		OverridesDir: *overridesDir,
	}

	if err := codegen.Generate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	girDir := fs.String("gir-dir", "gir-files", "Directory containing GIR XML files")
	outputDir := fs.String("output-dir", ".", "Root output directory (containing gst/ subdirectory)")
	overridesDir := fs.String("overrides-dir", "overrides", "Directory containing override YAML files")
	fs.Parse(args)

	cfg := &codegen.Config{
		GIRDir:       *girDir,
		OutputDir:    *outputDir,
		OverridesDir: *overridesDir,
	}

	if err := codegen.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("All generated files are up to date.")
}
