package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/typeconv"
)

// generateFunctions generates the _gen_functions.go file with standalone functions
// and constructors.
func generateFunctions(ctx *PackageContext) error {
	ns := ctx.Namespace

	var sections []string

	// Namespace-level functions.
	funcs := sortedFunctions(ns.Functions)
	for _, fn := range funcs {
		if ctx.Overrides.IsFunctionSkipped(fn.Name) {
			continue
		}
		if fn.CIdentifier == "" || fn.Deprecated != "" || fn.Introspectable == "0" {
			continue
		}
		code := generateStandaloneFunction(ctx, fn)
		if code != "" {
			sections = append(sections, code)
		}
	}

	// Constructors from GObject classes.
	for _, cls := range sortedClassesByName(ns.Classes) {
		if ctx.Overrides.IsTypeSkipped(cls.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		if rt == nil || rt.Kind != girparser.TypeKindGObject {
			continue
		}
		for _, ctor := range cls.Constructors {
			if ctx.Overrides.IsFunctionSkipped(ctor.Name) {
				continue
			}
			if ctor.CIdentifier == "" || ctor.Deprecated != "" || ctor.Introspectable == "0" {
				continue
			}
			code := generateConstructor(ctx, cls.Name, ctor, true)
			if code != "" {
				sections = append(sections, code)
			}
		}
	}

	// Constructors from MiniObject records.
	for _, rec := range sortedRecordsByName(ns.Records) {
		if ctx.Overrides.IsTypeSkipped(rec.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + rec.Name)
		if rt == nil || rt.Kind != girparser.TypeKindMiniObject {
			continue
		}
		for _, ctor := range rec.Constructors {
			if ctx.Overrides.IsFunctionSkipped(ctor.Name) {
				continue
			}
			if ctor.CIdentifier == "" || ctor.Deprecated != "" || ctor.Introspectable == "0" {
				continue
			}
			code := generateConstructor(ctx, rec.Name, ctor, false)
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

	// Collect all needed imports by scanning generated code.
	imports := detectImports(ctx, sections)
	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString("\t" + imp + "\n")
		}
		buf.WriteString(")\n\n")
	}

	for _, section := range sections {
		buf.WriteString(section)
	}

	outPath := ctx.genFilePath("_gen_functions.go")
	return writeFileIfChanged(outPath, buf.String())
}

// generateStandaloneFunction generates a package-level Go function.
func generateStandaloneFunction(ctx *PackageContext, fn girparser.Function) string {
	goFuncName := CSymbolToFuncName(fn.CIdentifier, NSPrefixLower(ctx.Package.Namespace))

	retInfo := analyzeReturn(ctx, fn.ReturnValue)

	var params []paramInfo
	hasError := fn.Throws
	if fn.Parameters != nil {
		for _, p := range fn.Parameters.Params {
			if p.Varargs != nil {
				return "" // Skip varargs functions.
			}
			pi := analyzeParam(ctx, p)
			if pi.isError {
				hasError = true
				continue
			}
			if pi.isCallback || pi.isInOut {
				return "" // Skip functions with callback or inout params.
			}
			params = append(params, pi)
		}
	}

	// Skip functions that return only void and have no error (nothing useful).
	if retInfo.isVoid && !hasError && len(params) == 0 {
		return ""
	}

	var buf strings.Builder

	// Doc comment.
	if fn.Doc != nil && fn.Doc.Text != "" {
		buf.WriteString(fmt.Sprintf("// %s: %s\n", goFuncName, singleLineDoc(fn.Doc.Text)))
	}

	// Signature.
	buf.WriteString(fmt.Sprintf("func %s(", goFuncName))
	first := true
	for _, p := range params {
		if p.isOut {
			continue
		}
		if !first {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s %s", p.name, p.goType))
		first = false
	}
	buf.WriteString(")")

	var retTypes []string
	if !retInfo.isVoid {
		retTypes = append(retTypes, retInfo.goType)
	}
	if hasError {
		retTypes = append(retTypes, "error")
	}
	if len(retTypes) == 1 {
		buf.WriteString(" " + retTypes[0])
	} else if len(retTypes) > 1 {
		buf.WriteString(" (" + strings.Join(retTypes, ", ") + ")")
	}

	buf.WriteString(" {\n")

	// Convert parameters.
	for _, p := range params {
		if p.cConvert != "" {
			buf.WriteString(p.cConvert)
		}
		if p.defers != "" {
			buf.WriteString(p.defers)
		}
	}

	if hasError {
		buf.WriteString("\tvar gerr *C.GError\n")
	}

	// Build C call.
	var cArgs []string
	for _, p := range params {
		if !p.isOut {
			cArgs = append(cArgs, p.cName)
		}
	}
	if hasError {
		cArgs = append(cArgs, "&gerr")
	}

	callExpr := fmt.Sprintf("C.%s(%s)", fn.CIdentifier, strings.Join(cArgs, ", "))

	if retInfo.isVoid && !hasError {
		buf.WriteString(fmt.Sprintf("\t%s\n", callExpr))
	} else if retInfo.isVoid && hasError {
		buf.WriteString(fmt.Sprintf("\t%s\n", callExpr))
		buf.WriteString(writeErrorCheck(""))
	} else {
		buf.WriteString(fmt.Sprintf("\tcResult := %s\n", callExpr))
		if hasError {
			buf.WriteString(writeErrorCheck(retInfo.goType))
		}
		writeReturnConversion(&buf, retInfo)
	}

	buf.WriteString("}\n\n")
	return buf.String()
}

// generateConstructor generates a NewXxx function from a GIR constructor.
func generateConstructor(ctx *PackageContext, typeName string, ctor girparser.Function, isGObject bool) string {
	goType := GIRNameToGoType(typeName)
	goFuncName := ConstructorName(goType, ctor.Name)

	retInfo := returnInfo{
		goType:    "*" + goType,
		isGObject: isGObject,
		isMiniObj: !isGObject,
		transfer:  typeconv.TransferFull, // Constructors always return transfer-full.
		nullable:  true,
		wrapFunc:  "FromGst" + goType + "UnsafeFull",
	}

	var params []paramInfo
	hasError := ctor.Throws
	if ctor.Parameters != nil {
		for _, p := range ctor.Parameters.Params {
			if p.Varargs != nil {
				return ""
			}
			pi := analyzeParam(ctx, p)
			if pi.isError {
				hasError = true
				continue
			}
			if pi.isCallback || pi.isInOut {
				return "" // Skip constructors with callback or inout params.
			}
			params = append(params, pi)
		}
	}

	var buf strings.Builder

	// Doc comment.
	if ctor.Doc != nil && ctor.Doc.Text != "" {
		buf.WriteString(fmt.Sprintf("// %s: %s\n", goFuncName, singleLineDoc(ctor.Doc.Text)))
	}

	// Signature.
	buf.WriteString(fmt.Sprintf("func %s(", goFuncName))
	first := true
	for _, p := range params {
		if p.isOut {
			continue
		}
		if !first {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s %s", p.name, p.goType))
		first = false
	}
	buf.WriteString(")")

	var retTypes []string
	retTypes = append(retTypes, retInfo.goType)
	if hasError {
		retTypes = append(retTypes, "error")
	}
	if len(retTypes) == 1 {
		buf.WriteString(" " + retTypes[0])
	} else {
		buf.WriteString(" (" + strings.Join(retTypes, ", ") + ")")
	}

	buf.WriteString(" {\n")

	// Convert parameters.
	for _, p := range params {
		if p.cConvert != "" {
			buf.WriteString(p.cConvert)
		}
		if p.defers != "" {
			buf.WriteString(p.defers)
		}
	}

	if hasError {
		buf.WriteString("\tvar gerr *C.GError\n")
	}

	// Build C call.
	var cArgs []string
	for _, p := range params {
		if !p.isOut {
			cArgs = append(cArgs, p.cName)
		}
	}
	if hasError {
		cArgs = append(cArgs, "&gerr")
	}

	callExpr := fmt.Sprintf("C.%s(%s)", ctor.CIdentifier, strings.Join(cArgs, ", "))
	buf.WriteString(fmt.Sprintf("\tcResult := %s\n", callExpr))

	if hasError {
		buf.WriteString(writeErrorCheck(retInfo.goType))
	}

	buf.WriteString(fmt.Sprintf("\treturn %s(unsafe.Pointer(cResult))\n", retInfo.wrapFunc))
	buf.WriteString("}\n\n")

	return buf.String()
}

func writeReturnConversion(buf *strings.Builder, ret returnInfo) {
	switch {
	case ret.isString:
		if ret.transfer == typeconv.TransferFull {
			buf.WriteString("\tdefer C.g_free((C.gpointer)(unsafe.Pointer(cResult)))\n")
		}
		buf.WriteString("\treturn C.GoString(cResult)\n")
	case ret.isBool:
		buf.WriteString("\treturn int(cResult) > 0\n")
	case ret.isGObject || ret.isMiniObj:
		if ret.nullable {
			buf.WriteString("\tif cResult == nil {\n\t\treturn nil\n\t}\n")
		}
		buf.WriteString(fmt.Sprintf("\treturn %s(unsafe.Pointer(cResult))\n", ret.wrapFunc))
	case ret.isEnum:
		buf.WriteString(fmt.Sprintf("\treturn %s(cResult)\n", ret.goType))
	case ret.isPrimitive:
		buf.WriteString(fmt.Sprintf("\treturn %s(cResult)\n", ret.goType))
	default:
		buf.WriteString("\treturn cResult\n")
	}
}

func sortedFunctions(funcs []girparser.Function) []girparser.Function {
	sorted := make([]girparser.Function, len(funcs))
	copy(sorted, funcs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
