package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/apidiff"
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
	case "check-api":
		cmdCheckAPI(os.Args[2:])
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
	fmt.Fprintf(os.Stderr, "  check-api   Compare API surfaces between old and new generated files\n")
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

func cmdCheckAPI(args []string) {
	fs := flag.NewFlagSet("check-api", flag.ExitOnError)
	oldDir := fs.String("old-dir", "", "Root directory with old generated files (containing gst/ subdirectory)")
	newDir := fs.String("new-dir", "", "Root directory with new generated files (containing gst/ subdirectory)")
	oldVersion := fs.String("old-version", "", "Old GStreamer version (e.g., 1.24)")
	newVersion := fs.String("new-version", "", "New GStreamer version (e.g., 1.26)")
	format := fs.String("format", "text", "Output format: text, markdown, json")
	fs.Parse(args)

	if *oldDir == "" || *newDir == "" {
		fmt.Fprintf(os.Stderr, "error: --old-dir and --new-dir are required\n")
		os.Exit(1)
	}

	var reports []*apidiff.DiffReport
	for _, pkg := range codegen.DefaultPackages {
		oldPkg := filepath.Join(*oldDir, pkg.GoPackage)
		newPkg := filepath.Join(*newDir, pkg.GoPackage)

		// Skip if neither directory exists.
		oldExists := dirExists(oldPkg)
		newExists := dirExists(newPkg)
		if !oldExists && !newExists {
			continue
		}

		var oldAPI, newAPI *apidiff.PackageAPI
		var err error

		if oldExists {
			oldAPI, err = apidiff.ExtractPackageAPI(oldPkg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error extracting old API for %s: %v\n", pkg.GoPackage, err)
				os.Exit(1)
			}
		} else {
			oldAPI = apidiff.NewPackageAPI(filepath.Base(pkg.GoPackage))
		}

		if newExists {
			newAPI, err = apidiff.ExtractPackageAPI(newPkg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error extracting new API for %s: %v\n", pkg.GoPackage, err)
				os.Exit(1)
			}
		} else {
			newAPI = apidiff.NewPackageAPI(filepath.Base(pkg.GoPackage))
		}

		report := apidiff.Diff(oldAPI, newAPI)
		report.Package = pkg.GoPackage
		reports = append(reports, report)
	}

	switch *format {
	case "text":
		fmt.Print(apidiff.FormatText(reports))
	case "markdown":
		fmt.Print(apidiff.FormatMarkdown(reports))
	case "json":
		data, err := apidiff.FormatJSON(reports, *oldVersion, *newVersion)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error formatting JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", *format)
		os.Exit(1)
	}

	if apidiff.HasBreakingChanges(reports) {
		os.Exit(2) // Exit code 2 signals breaking changes detected.
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
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
