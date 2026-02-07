package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
)

// generateCallbacks generates the _gen_callbacks.go file with callback type definitions
// and signal connection methods.
func generateCallbacks(ctx *PackageContext) error {
	ns := ctx.Namespace

	var sections []string

	// Generate callback type definitions.
	callbacks := sortedCallbacks(ns.Callbacks)
	for _, cb := range callbacks {
		if ctx.Overrides.IsTypeSkipped(cb.Name) {
			continue
		}
		code := generateCallbackType(ctx, cb)
		if code != "" {
			sections = append(sections, code)
		}
	}

	// Generate signal Connect methods for GObject classes.
	for _, cls := range sortedClassesByName(ns.Classes) {
		if ctx.Overrides.IsTypeSkipped(cls.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		if rt == nil || rt.Kind != girparser.TypeKindGObject {
			continue
		}
		for _, sig := range cls.Signals {
			if ctx.Overrides.IsMethodSkipped(cls.Name, "signal::"+sig.Name) {
				continue
			}
			code := generateSignalConnect(ctx, cls.Name, sig)
			if code != "" {
				sections = append(sections, code)
			}
		}
	}

	if len(sections) == 0 {
		return nil
	}

	var buf strings.Builder

	writeGeneratedHeader(&buf, ctx)
	buf.WriteString("package ")
	buf.WriteString(goPackageName(ctx.Package.GoPackage))
	buf.WriteString("\n\n")

	buf.WriteString("/*\n#include \"gst.go.h\"\n*/\nimport \"C\"\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/go-gst/go-glib/glib\"\n")
	buf.WriteString(")\n\n")

	for _, section := range sections {
		buf.WriteString(section)
	}

	outPath := ctx.genFilePath("_gen_callbacks.go")
	return writeFileIfChanged(outPath, buf.String())
}

// generateCallbackType generates a Go type definition for a GIR callback.
func generateCallbackType(ctx *PackageContext, cb girparser.Callback) string {
	goType := GIRNameToGoType(cb.Name)

	// Build the parameter list.
	var goParams []string
	if cb.Parameters != nil {
		for _, p := range cb.Parameters.Params {
			if p.Varargs != nil {
				return "" // Skip varargs callbacks.
			}
			pi := analyzeParam(ctx, p)
			goParams = append(goParams, fmt.Sprintf("%s %s", pi.name, pi.goType))
		}
	}

	// Build return type.
	retInfo := analyzeReturn(ctx, cb.ReturnValue)
	retStr := ""
	if !retInfo.isVoid {
		retStr = " " + retInfo.goType
	}

	var buf strings.Builder
	if cb.Doc != nil && cb.Doc.Text != "" {
		writeDocComment(&buf, goType, cb.Doc.Text)
	}
	buf.WriteString(fmt.Sprintf("type %s func(%s)%s\n\n", goType, strings.Join(goParams, ", "), retStr))

	return buf.String()
}

// generateSignalConnect generates a Connect method for a GObject signal.
func generateSignalConnect(ctx *PackageContext, typeName string, sig girparser.Signal) string {
	goType := GIRNameToGoType(typeName)
	receiver := GoTypeToReceiverName(goType)

	// Convert signal name to Go method name: "pad-added" → "ConnectPadAdded"
	signalGoName := "Connect" + UnderscoreToPascal(strings.ReplaceAll(sig.Name, "-", "_"))

	// Build the callback parameter types.
	var cbParams []string
	var unwrapLines []string
	if sig.Parameters != nil {
		for i, p := range sig.Parameters.Params {
			pi := analyzeParam(ctx, p)
			cbParams = append(cbParams, fmt.Sprintf("%s %s", pi.name, pi.goType))

			// For GObject params, unwrap from interface{} values.
			switch {
			case strings.HasPrefix(pi.goType, "*"):
				unwrapLines = append(unwrapLines, fmt.Sprintf(
					"\t\t%s := values[%d].(%s)", pi.name, i, pi.goType))
			default:
				unwrapLines = append(unwrapLines, fmt.Sprintf(
					"\t\t%s := values[%d].(%s)", pi.name, i, pi.goType))
			}
		}
	}

	// Return type of the callback.
	retInfo := analyzeReturn(ctx, sig.ReturnValue)

	var buf strings.Builder

	if sig.Doc != nil && sig.Doc.Text != "" {
		buf.WriteString(fmt.Sprintf("// %s: %s\n", signalGoName, singleLineDoc(sig.Doc.Text)))
	}

	// For simplicity, generate Connect methods that use glib.Connect with signal name.
	buf.WriteString(fmt.Sprintf("func (%s *%s) %s(f func(%s)", receiver, goType, signalGoName,
		strings.Join(cbParams, ", ")))
	if !retInfo.isVoid {
		buf.WriteString(fmt.Sprintf(" %s", retInfo.goType))
	}
	buf.WriteString(") glib.SignalHandle {\n")

	buf.WriteString(fmt.Sprintf("\treturn %s.Connect(\"%s\", f)\n", receiver, sig.Name))
	buf.WriteString("}\n\n")

	return buf.String()
}

func sortedCallbacks(callbacks []girparser.Callback) []girparser.Callback {
	sorted := make([]girparser.Callback, len(callbacks))
	copy(sorted, callbacks)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
