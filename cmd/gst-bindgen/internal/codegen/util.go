package codegen

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// genFilePath returns the full path for a generated file in the package output directory.
func (ctx *PackageContext) genFilePath(filename string) string {
	return filepath.Join(ctx.OutputDir, filename)
}

// writeFileIfChanged writes content to a file, only if the content differs from what's on disk.
func writeFileIfChanged(path string, content string) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == content {
		return nil // No change needed.
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// goModuleRoot is the base Go module path for the project.
const goModuleRoot = "github.com/go-gst/go-gst"

// isKnownNamespace returns true if the namespace has a corresponding Go package
// in our DefaultPackages list.
func isKnownNamespace(ns string) bool {
	for _, pkg := range DefaultPackages {
		if pkg.Namespace == ns {
			return true
		}
	}
	return false
}

// detectImports scans generated code sections for package references and returns
// a sorted list of import strings (e.g., `"fmt"`, `"unsafe"`, `base "github.com/..."`)
func detectImports(ctx *PackageContext, sections []string) []string {
	allCode := strings.Join(sections, "\n")

	needed := make(map[string]string) // alias → import path

	// Standard library imports.
	if strings.Contains(allCode, "fmt.Errorf") || strings.Contains(allCode, "fmt.Sprintf") {
		needed["fmt"] = `"fmt"`
	}
	if strings.Contains(allCode, "unsafe.Pointer") || strings.Contains(allCode, "C.free") ||
		strings.Contains(allCode, "C.g_free") {
		needed["unsafe"] = `"unsafe"`
	}

	// Cross-package imports: detect references like "base.", "video.", "audio.", etc.
	for _, pkg := range DefaultPackages {
		if pkg.Namespace == ctx.Package.Namespace {
			continue // Same package, no import needed.
		}
		alias := goPackageName(pkg.GoPackage)
		if strings.Contains(allCode, alias+".") {
			importPath := goModuleRoot + "/" + pkg.GoPackage
			needed[alias] = alias + ` "` + importPath + `"`
		}
	}

	// Sort imports: standard library first, then project packages.
	var stdImports, pkgImports []string
	for _, imp := range needed {
		if strings.HasPrefix(imp, `"`) && !strings.Contains(imp, goModuleRoot) {
			stdImports = append(stdImports, imp)
		} else {
			pkgImports = append(pkgImports, imp)
		}
	}
	sort.Strings(stdImports)
	sort.Strings(pkgImports)

	result := stdImports
	if len(stdImports) > 0 && len(pkgImports) > 0 {
		result = append(result, "") // Blank line between groups.
	}
	result = append(result, pkgImports...)
	return result
}
