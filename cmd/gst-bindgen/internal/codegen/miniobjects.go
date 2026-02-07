package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
)

// generateMiniObjects generates the _gen_miniobjects.go file with mini-object type scaffolding.
func generateMiniObjects(ctx *PackageContext) error {
	ns := ctx.Namespace

	// Collect mini-object records.
	var records []girparser.Record
	for _, rec := range ns.Records {
		if ctx.Overrides.IsTypeSkipped(rec.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + rec.Name)
		if rt == nil || rt.Kind != girparser.TypeKindMiniObject {
			continue
		}
		records = append(records, rec)
	}

	if len(records) == 0 {
		return nil
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})

	var buf strings.Builder

	writeGeneratedHeader(&buf, ctx)
	buf.WriteString("package ")
	buf.WriteString(goPackageName(ctx.Package.GoPackage))
	buf.WriteString("\n\n")

	buf.WriteString("/*\n#include \"gst.go.h\"\n*/\nimport \"C\"\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"runtime\"\n")
	buf.WriteString("\t\"unsafe\"\n")
	buf.WriteString(")\n\n")

	for _, rec := range records {
		writeMiniObjectType(&buf, ctx, rec)
	}

	outPath := ctx.genFilePath("_gen_miniobjects.go")
	return writeFileIfChanged(outPath, buf.String())
}

// writeMiniObjectType writes struct, FromUnsafe*, Instance, Ref, Unref for a mini-object.
func writeMiniObjectType(buf *strings.Builder, ctx *PackageContext, rec girparser.Record) {
	goType := GIRNameToGoType(rec.Name)
	receiver := GoTypeToReceiverName(goType)
	cType := rec.CType
	if cType == "" {
		cType = "Gst" + rec.Name
	}

	// Determine the pointer field name — use a lowered version of the type name.
	ptrField := "ptr"

	// Check overrides for extra fields.
	var extraFields []string
	if fields, ok := ctx.Overrides.MiniObjectExtraFields[rec.Name]; ok {
		for _, f := range fields {
			extraFields = append(extraFields, fmt.Sprintf("\t%s %s", f.Name, f.Type))
		}
	}

	// Doc comment.
	if rec.Doc != nil && rec.Doc.Text != "" {
		writeDocComment(buf, goType, rec.Doc.Text)
	}

	// Struct definition.
	buf.WriteString(fmt.Sprintf("type %s struct {\n", goType))
	buf.WriteString(fmt.Sprintf("\t%s *C.%s\n", ptrField, cType))
	for _, ef := range extraFields {
		buf.WriteString(ef)
		buf.WriteString("\n")
	}
	buf.WriteString("}\n\n")

	// wrapXxx function.
	buf.WriteString(fmt.Sprintf("func wrap%s(ptr *C.%s) *%s { return &%s{%s: ptr} }\n\n",
		goType, cType, goType, goType, ptrField))

	// FromGstXxxUnsafeNone — refs the object, sets finalizer.
	buf.WriteString(fmt.Sprintf("// FromGst%sUnsafeNone wraps with transfer-none semantics (adds a ref).\n", goType))
	buf.WriteString(fmt.Sprintf("func FromGst%sUnsafeNone(ptr unsafe.Pointer) *%s {\n", goType, goType))
	buf.WriteString("\tif ptr == nil {\n\t\treturn nil\n\t}\n")
	buf.WriteString(fmt.Sprintf("\t%s := wrap%s(C.toGst%s(ptr))\n", receiver, goType, rec.Name))
	buf.WriteString(fmt.Sprintf("\t%s.Ref()\n", receiver))
	buf.WriteString(fmt.Sprintf("\truntime.SetFinalizer(%s, (*%s).Unref)\n", receiver, goType))
	buf.WriteString(fmt.Sprintf("\treturn %s\n", receiver))
	buf.WriteString("}\n\n")

	// FromGstXxxUnsafeFull — takes ownership, sets finalizer only.
	buf.WriteString(fmt.Sprintf("// FromGst%sUnsafeFull wraps with transfer-full semantics.\n", goType))
	buf.WriteString(fmt.Sprintf("func FromGst%sUnsafeFull(ptr unsafe.Pointer) *%s {\n", goType, goType))
	buf.WriteString("\tif ptr == nil {\n\t\treturn nil\n\t}\n")
	buf.WriteString(fmt.Sprintf("\t%s := wrap%s(C.toGst%s(ptr))\n", receiver, goType, rec.Name))
	buf.WriteString(fmt.Sprintf("\truntime.SetFinalizer(%s, (*%s).Unref)\n", receiver, goType))
	buf.WriteString(fmt.Sprintf("\treturn %s\n", receiver))
	buf.WriteString("}\n\n")

	// Instance() method.
	buf.WriteString(fmt.Sprintf("// Instance returns the native C %s pointer.\n", cType))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Instance() *C.%s { return C.toGst%s(unsafe.Pointer(%s.%s)) }\n\n",
		receiver, goType, cType, rec.Name, receiver, ptrField))

	// Ref() method.
	lowerName := strings.ToLower(cType)
	// Use CSymbolPrefix if available, otherwise derive from CType.
	refFunc := deriveRefFunc(rec, cType)
	unrefFunc := deriveUnrefFunc(rec, cType)

	buf.WriteString(fmt.Sprintf("// Ref increases the reference count on the %s.\n", goType))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Ref() *%s {\n", receiver, goType, goType))
	buf.WriteString(fmt.Sprintf("\treturn wrap%s(C.%s(%s.Instance()))\n", goType, refFunc, receiver))
	buf.WriteString("}\n\n")

	_ = lowerName // suppress unused

	// Unref() method.
	buf.WriteString(fmt.Sprintf("// Unref decreases the reference count on the %s.\n", goType))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Unref() {\n", receiver, goType))
	buf.WriteString(fmt.Sprintf("\tC.%s(%s.Instance())\n", unrefFunc, receiver))
	buf.WriteString("}\n\n")
}

// deriveRefFunc returns the C ref function name for a mini-object.
func deriveRefFunc(rec girparser.Record, cType string) string {
	// Look for a method named "ref" on the record.
	for _, m := range rec.Methods {
		if m.Name == "ref" && m.CIdentifier != "" {
			return m.CIdentifier
		}
	}
	// Derive from the C symbol prefix: gst_<symbol_prefix>_ref
	if rec.CSymbolPrefix != "" {
		return "gst_" + rec.CSymbolPrefix + "_ref"
	}
	// Fall back to gst_mini_object_ref (generic)
	return "gst_mini_object_ref"
}

// deriveUnrefFunc returns the C unref function name for a mini-object.
func deriveUnrefFunc(rec girparser.Record, cType string) string {
	for _, m := range rec.Methods {
		if m.Name == "unref" && m.CIdentifier != "" {
			return m.CIdentifier
		}
	}
	if rec.CSymbolPrefix != "" {
		return "gst_" + rec.CSymbolPrefix + "_unref"
	}
	return "gst_mini_object_unref"
}
