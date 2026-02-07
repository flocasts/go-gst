package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/typeconv"
)

// generateMethods generates the _gen_methods.go file with Go methods for GObject and MiniObject types.
func generateMethods(ctx *PackageContext) error {
	ns := ctx.Namespace

	var sections []string

	// Collect methods from GObject classes.
	for _, cls := range sortedClassesByName(ns.Classes) {
		if ctx.Overrides.IsTypeSkipped(cls.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		if rt == nil || rt.Kind != girparser.TypeKindGObject {
			continue
		}
		if len(cls.Methods) == 0 {
			continue
		}
		section := generateMethodsForType(ctx, cls.Name, cls.Methods, true)
		if section != "" {
			sections = append(sections, section)
		}
	}

	// Collect methods from MiniObject records.
	for _, rec := range sortedRecordsByName(ns.Records) {
		if ctx.Overrides.IsTypeSkipped(rec.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + rec.Name)
		if rt == nil || rt.Kind != girparser.TypeKindMiniObject {
			continue
		}
		if len(rec.Methods) == 0 {
			continue
		}
		section := generateMethodsForType(ctx, rec.Name, rec.Methods, false)
		if section != "" {
			sections = append(sections, section)
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

	outPath := ctx.genFilePath("_gen_methods.go")
	return writeFileIfChanged(outPath, buf.String())
}

// generateMethodsForType generates method code for a single type.
func generateMethodsForType(ctx *PackageContext, typeName string, methods []girparser.Method, isGObject bool) string {
	goType := GIRNameToGoType(typeName)
	receiver := GoTypeToReceiverName(goType)

	var buf strings.Builder

	for _, method := range methods {
		if ctx.Overrides.IsMethodSkipped(typeName, method.Name) {
			continue
		}
		if method.CIdentifier == "" {
			continue
		}
		// Skip deprecated methods.
		if method.Deprecated != "" {
			continue
		}
		// Skip non-introspectable methods.
		if method.Introspectable == "0" {
			continue
		}

		code := generateSingleMethod(ctx, goType, receiver, method, isGObject)
		if code != "" {
			buf.WriteString(code)
		}
	}

	return buf.String()
}

// generateSingleMethod generates Go code for a single method.
func generateSingleMethod(ctx *PackageContext, goType, receiver string, method girparser.Method, isGObject bool) string {
	goMethodName := CSymbolToMethodName(method.CIdentifier, methodPrefix(ctx, goType))

	// Parse return type.
	retInfo := analyzeReturn(ctx, method.ReturnValue)

	// Parse parameters.
	var params []paramInfo
	hasError := false
	if method.Parameters != nil {
		for _, p := range method.Parameters.Params {
			if p.Varargs != nil {
				return "" // Skip varargs methods.
			}
			pi := analyzeParam(ctx, p)
			if pi.isError {
				hasError = true
				continue
			}
			if pi.isCallback || pi.isInOut {
				return "" // Skip methods with callback or inout params (need special handling).
			}
			params = append(params, pi)
		}
	}

	var buf strings.Builder

	// Doc comment.
	if method.Doc != nil && method.Doc.Text != "" {
		buf.WriteString(fmt.Sprintf("// %s: %s\n", goMethodName, singleLineDoc(method.Doc.Text)))
	}

	// Signature.
	sig := buildMethodSignature(goType, receiver, goMethodName, params, retInfo, hasError)
	buf.WriteString(sig)
	buf.WriteString(" {\n")

	// Body: convert parameters, call C function, convert return.
	body := buildMethodBody(ctx, receiver, method.CIdentifier, params, retInfo, hasError, isGObject)
	buf.WriteString(body)

	buf.WriteString("}\n\n")

	return buf.String()
}

// paramInfo describes a parsed method parameter.
type paramInfo struct {
	name       string // Go parameter name
	goType     string // Go type
	cConvert   string // Code to convert Go → C
	cName      string // C variable name to pass to the C call
	defers     string // defer statement if needed
	isError    bool   // GError** parameter
	isOut      bool   // output parameter
	isCallback bool   // callback parameter (can't pass Go func to C)
	isInOut    bool   // inout parameter (double pointer, complex handling)
}

// returnInfo describes a parsed return value.
type returnInfo struct {
	goType      string // Go return type
	isVoid      bool
	isString    bool
	isGObject   bool
	isMiniObj   bool
	isBool      bool
	isPrimitive bool
	isEnum      bool
	transfer    typeconv.TransferOwnership
	nullable    bool
	wrapFunc    string // e.g., "FromGstElementUnsafeFull"
	cType       string // C type
}

func analyzeReturn(ctx *PackageContext, rv *girparser.ReturnValue) returnInfo {
	if rv == nil || rv.Type == nil || rv.Type.Name == "none" {
		return returnInfo{isVoid: true}
	}

	transfer := typeconv.TransferOwnership(rv.TransferOwnership)
	info := returnInfo{
		transfer: transfer,
		nullable: rv.Nullable,
	}

	typeName := rv.Type.Name
	cType := rv.Type.CType

	// Check primitives.
	if gt, ok := typeconv.PrimitiveTypeMap[typeName]; ok {
		info.goType = gt.Name
		info.isPrimitive = true
		info.cType = cType
		if typeName == "utf8" || typeName == "filename" {
			info.isString = true
		}
		if typeName == "gboolean" {
			info.isBool = true
		}
		return info
	}

	// Look up in type registry.
	qualName := ctx.Registry.QualifyName(ctx.Package.Namespace, typeName)
	rt := ctx.Registry.LookupType(qualName)
	if rt != nil {
		crossPkg := rt.Namespace != ctx.Package.Namespace

		// Types from unknown namespaces (GLib, GObject, etc.) → treat as void.
		if crossPkg && !isKnownNamespace(rt.Namespace) {
			info.isVoid = true
			return info
		}

		goType := GIRNameToGoType(rt.Name)

		// Cross-package reference.
		if crossPkg {
			pkgAlias := goPackageName(namespaceToPkgPath(rt.Namespace))
			goType = pkgAlias + "." + goType
		}

		switch rt.Kind {
		case girparser.TypeKindGObject:
			info.goType = "*" + goType
			info.isGObject = true
			info.wrapFunc = typeconv.ReturnWrapping(rt.Name, transfer, false)
			if crossPkg {
				info.wrapFunc = goPackageName(namespaceToPkgPath(rt.Namespace)) + "." + info.wrapFunc
			}
		case girparser.TypeKindMiniObject:
			info.goType = "*" + goType
			info.isMiniObj = true
			info.wrapFunc = typeconv.ReturnWrapping(rt.Name, transfer, true)
			if crossPkg {
				info.wrapFunc = goPackageName(namespaceToPkgPath(rt.Namespace)) + "." + info.wrapFunc
			}
		case girparser.TypeKindEnum, girparser.TypeKindBitfield:
			info.goType = goType
			info.isEnum = true
		case girparser.TypeKindCallback:
			// Callback return types are opaque — treat as void for now.
			info.isVoid = true
			return info
		default:
			// Plain records, interfaces, aliases from external namespaces → treat as void.
			if crossPkg {
				info.isVoid = true
				return info
			}
			info.goType = goType
		}
		info.cType = cType
		return info
	}

	// Unknown type — skip.
	info.isVoid = true
	return info
}

func analyzeParam(ctx *PackageContext, p girparser.Parameter) paramInfo {
	info := paramInfo{
		name:    sanitizeGoName(p.Name),
		isOut:   p.Direction == "out",
		isInOut: p.Direction == "inout",
	}

	// GError** parameter.
	if p.Type != nil && p.Type.Name == "GLib.Error" {
		info.isError = true
		return info
	}

	// Array or no-type parameters → unsafe.Pointer pass-through.
	if p.Type == nil {
		info.goType = "unsafe.Pointer"
		info.cName = info.name
		return info
	}

	typeName := p.Type.Name
	cType := cleanCType(p.Type.CType)

	// Primitives.
	if gt, ok := typeconv.PrimitiveTypeMap[typeName]; ok {
		info.goType = gt.Name
		_, hasConv := typeconv.PrimitiveConversions[typeName]
		if hasConv {
			cVarName := "c" + capitalizeFirst(info.name)
			switch typeName {
			case "utf8", "filename":
				info.cName = cVarName
				info.cConvert = fmt.Sprintf("\t%s := C.CString(%s)\n", cVarName, info.name)
				info.defers = fmt.Sprintf("\tdefer C.free(unsafe.Pointer(%s))\n", cVarName)
			case "gboolean":
				info.cName = "gboolean(" + info.name + ")"
			default:
				info.cName = fmt.Sprintf("C.%s(%s)", cTypeForGIR(typeName), info.name)
			}
		} else {
			info.cName = fmt.Sprintf("C.%s(%s)", cTypeForGIR(typeName), info.name)
		}
		return info
	}

	// Type registry lookup.
	qualName := ctx.Registry.QualifyName(ctx.Package.Namespace, typeName)
	rt := ctx.Registry.LookupType(qualName)
	if rt != nil {
		crossPkg := rt.Namespace != ctx.Package.Namespace

		// Types from unknown namespaces (GLib, GObject, etc.) → treat as opaque.
		if crossPkg && !isKnownNamespace(rt.Namespace) {
			info.goType = "unsafe.Pointer"
			info.cName = info.name
			return info
		}

		goType := GIRNameToGoType(rt.Name)
		if crossPkg {
			pkgAlias := goPackageName(namespaceToPkgPath(rt.Namespace))
			goType = pkgAlias + "." + goType
		}

		switch rt.Kind {
		case girparser.TypeKindGObject, girparser.TypeKindMiniObject:
			info.goType = "*" + goType
			info.cName = info.name + ".Instance()"
		case girparser.TypeKindEnum, girparser.TypeKindBitfield:
			info.goType = goType
			enumCType := cType
			if enumCType == "" {
				enumCType = rt.CType
			}
			info.cName = fmt.Sprintf("C.%s(%s)", enumCType, info.name)
		case girparser.TypeKindCallback:
			info.isCallback = true
			info.goType = goType
			info.cName = info.name
		default:
			// Plain records, interfaces, aliases from external namespaces → unsafe.Pointer.
			if crossPkg {
				info.goType = "unsafe.Pointer"
				info.cName = info.name
			} else {
				info.goType = goType
				info.cName = info.name
			}
		}
		return info
	}

	// Unknown — use unsafe.Pointer.
	info.goType = "unsafe.Pointer"
	info.cName = info.name
	return info
}

// cleanCType strips qualifiers like "const " from a C type for use in C casts.
func cleanCType(cType string) string {
	cType = strings.TrimPrefix(cType, "const ")
	cType = strings.TrimSuffix(cType, "*")
	cType = strings.TrimSpace(cType)
	return cType
}

func buildMethodSignature(goType, receiver, methodName string, params []paramInfo, ret returnInfo, hasError bool) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("func (%s *%s) %s(", receiver, goType, methodName))

	first := true
	for _, p := range params {
		if p.isOut {
			continue // Skip output params in signature for now.
		}
		if !first {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%s %s", p.name, p.goType))
		first = false
	}
	buf.WriteString(")")

	// Return types.
	var retTypes []string
	if !ret.isVoid {
		retTypes = append(retTypes, ret.goType)
	}
	if hasError {
		retTypes = append(retTypes, "error")
	}

	if len(retTypes) == 1 {
		buf.WriteString(" " + retTypes[0])
	} else if len(retTypes) > 1 {
		buf.WriteString(" (" + strings.Join(retTypes, ", ") + ")")
	}

	return buf.String()
}

func buildMethodBody(ctx *PackageContext, receiver, cFunc string, params []paramInfo, ret returnInfo, hasError, isGObject bool) string {
	var buf strings.Builder

	// Convert parameters.
	for _, p := range params {
		if p.cConvert != "" {
			buf.WriteString(p.cConvert)
		}
		if p.defers != "" {
			buf.WriteString(p.defers)
		}
	}

	// Error variable.
	if hasError {
		buf.WriteString("\tvar gerr *C.GError\n")
	}

	// Build C call arguments.
	var cArgs []string
	cArgs = append(cArgs, receiver+".Instance()")
	for _, p := range params {
		if !p.isOut {
			cArgs = append(cArgs, p.cName)
		}
	}
	if hasError {
		cArgs = append(cArgs, "&gerr")
	}

	callExpr := fmt.Sprintf("C.%s(%s)", cFunc, strings.Join(cArgs, ", "))

	if ret.isVoid && !hasError {
		buf.WriteString(fmt.Sprintf("\t%s\n", callExpr))
		return buf.String()
	}

	if ret.isVoid && hasError {
		buf.WriteString(fmt.Sprintf("\t%s\n", callExpr))
		buf.WriteString(writeErrorCheck(""))
		return buf.String()
	}

	// Non-void return.
	buf.WriteString(fmt.Sprintf("\tcResult := %s\n", callExpr))

	if hasError {
		buf.WriteString(writeErrorCheck(ret.goType))
	}

	// Convert return value.
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

	return buf.String()
}

func writeErrorCheck(retType string) string {
	var buf strings.Builder
	buf.WriteString("\tif gerr != nil {\n")
	buf.WriteString("\t\tdefer C.g_error_free(gerr)\n")
	if retType != "" && strings.HasPrefix(retType, "*") {
		buf.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%%s\", C.GoString(gerr.message))\n"))
	} else if retType != "" {
		buf.WriteString(fmt.Sprintf("\t\tvar zero %s\n", retType))
		buf.WriteString(fmt.Sprintf("\t\treturn zero, fmt.Errorf(\"%%s\", C.GoString(gerr.message))\n"))
	} else {
		buf.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"%%s\", C.GoString(gerr.message))\n"))
	}
	buf.WriteString("\t}\n")
	return buf.String()
}

// methodPrefix returns the C symbol prefix for a method on a type.
// For example, for type "Element" in namespace "Gst", this returns "gst_element_".
func methodPrefix(ctx *PackageContext, goType string) string {
	// Look up the class to get CSymbolPrefix.
	cls := ctx.Registry.FindClassByName(ctx.Package.Namespace, goType)
	if cls != nil && cls.CSymbolPrefix != "" {
		return "gst_" + cls.CSymbolPrefix + "_"
	}
	// Look up the record.
	rec := ctx.Registry.FindRecordByName(ctx.Package.Namespace, goType)
	if rec != nil && rec.CSymbolPrefix != "" {
		return "gst_" + rec.CSymbolPrefix + "_"
	}
	// Fall back to converting the Go type.
	return NSPrefixLower(ctx.Package.Namespace) + strings.ToLower(goType) + "_"
}

func sanitizeGoName(name string) string {
	// Go reserved words.
	switch name {
	case "type", "func", "range", "map", "string", "int", "error", "default",
		"select", "case", "defer", "go", "chan", "interface", "struct", "var",
		"const", "break", "continue", "return", "package", "import":
		return name + "Val"
	}
	return name
}

func cTypeForGIR(girType string) string {
	switch girType {
	case "gint":
		return "gint"
	case "guint":
		return "guint"
	case "gint64":
		return "gint64"
	case "guint64":
		return "guint64"
	case "gfloat":
		return "gfloat"
	case "gdouble":
		return "gdouble"
	case "gboolean":
		return "gboolean"
	case "gsize":
		return "gsize"
	case "gssize":
		return "gssize"
	case "gint8":
		return "gint8"
	case "guint8":
		return "guint8"
	case "gint16":
		return "gint16"
	case "guint16":
		return "guint16"
	case "gint32":
		return "gint32"
	case "guint32":
		return "guint32"
	case "glong":
		return "glong"
	case "gulong":
		return "gulong"
	default:
		return girType
	}
}

func sortedClassesByName(classes []girparser.Class) []girparser.Class {
	sorted := make([]girparser.Class, len(classes))
	copy(sorted, classes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

func sortedRecordsByName(records []girparser.Record) []girparser.Record {
	sorted := make([]girparser.Record, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}
