package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
)

// generateMarshalers generates the _gen_marshalers.go file with GValue marshaling
// functions and an init() registration.
func generateMarshalers(ctx *PackageContext) error {
	ns := ctx.Namespace

	type marshalEntry struct {
		goType     string
		getTypeC   string // e.g., "gst_element_get_type()"
		isGObject  bool
		isMiniObj  bool
		cType      string
		ptrField   string // for mini-objects
	}

	var entries []marshalEntry

	// GObject classes.
	for _, cls := range ns.Classes {
		if ctx.Overrides.IsTypeSkipped(cls.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		if rt == nil || rt.Kind != girparser.TypeKindGObject {
			continue
		}
		if cls.GLibGetType == "" {
			continue
		}
		entries = append(entries, marshalEntry{
			goType:    GIRNameToGoType(cls.Name),
			getTypeC:  fmt.Sprintf("C.%s()", cls.GLibGetType),
			isGObject: true,
			cType:     cls.CType,
		})
	}

	// MiniObject records.
	for _, rec := range ns.Records {
		if ctx.Overrides.IsTypeSkipped(rec.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + rec.Name)
		if rt == nil || rt.Kind != girparser.TypeKindMiniObject {
			continue
		}
		if rec.GLibGetType == "" {
			continue
		}
		cType := rec.CType
		if cType == "" {
			cType = "Gst" + rec.Name
		}
		entries = append(entries, marshalEntry{
			goType:    GIRNameToGoType(rec.Name),
			getTypeC:  fmt.Sprintf("C.%s()", rec.GLibGetType),
			isMiniObj: true,
			cType:     cType,
			ptrField:  "ptr",
		})
	}

	if len(entries) == 0 {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].goType < entries[j].goType
	})

	var buf strings.Builder

	writeGeneratedHeader(&buf, ctx)
	buf.WriteString("package ")
	buf.WriteString(goPackageName(ctx.Package.GoPackage))
	buf.WriteString("\n\n")

	buf.WriteString("/*\n#include \"gst.go.h\"\n*/\nimport \"C\"\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"unsafe\"\n\n")
	buf.WriteString("\t\"github.com/go-gst/go-glib/glib\"\n")
	buf.WriteString(")\n\n")

	// init() function with registration.
	buf.WriteString("func init() { registerGeneratedMarshalers() }\n\n")
	buf.WriteString("func registerGeneratedMarshalers() {\n")
	buf.WriteString("\ttm := []glib.TypeMarshaler{\n")
	for _, e := range entries {
		buf.WriteString(fmt.Sprintf("\t\t{T: glib.Type(%s), F: marshal%s},\n", e.getTypeC, e.goType))
	}
	buf.WriteString("\t}\n")
	buf.WriteString("\tglib.RegisterGValueMarshalers(tm)\n")
	buf.WriteString("}\n\n")

	// Marshalers for GObject types.
	buf.WriteString("func toGValue(p unsafe.Pointer) *C.GValue {\n")
	buf.WriteString("\treturn (*C.GValue)(p)\n")
	buf.WriteString("}\n\n")

	for _, e := range entries {
		if e.isGObject {
			writeGObjectMarshaler(&buf, e.goType)
		} else if e.isMiniObj {
			writeMiniObjectMarshaler(&buf, e.goType, e.cType)
		}
	}

	outPath := ctx.genFilePath("_gen_marshalers.go")
	return writeFileIfChanged(outPath, buf.String())
}

func writeGObjectMarshaler(buf *strings.Builder, goType string) {
	buf.WriteString(fmt.Sprintf("func marshal%s(p unsafe.Pointer) (interface{}, error) {\n", goType))
	buf.WriteString("\tc := C.g_value_get_object(toGValue(p))\n")
	buf.WriteString("\tobj := &glib.Object{GObject: glib.ToGObject(unsafe.Pointer(c))}\n")
	buf.WriteString(fmt.Sprintf("\treturn wrap%s(obj), nil\n", goType))
	buf.WriteString("}\n\n")
}

func writeMiniObjectMarshaler(buf *strings.Builder, goType, cType string) {
	buf.WriteString(fmt.Sprintf("func marshal%s(p unsafe.Pointer) (interface{}, error) {\n", goType))
	buf.WriteString("\tc := C.g_value_get_boxed(toGValue(p))\n")
	buf.WriteString(fmt.Sprintf("\treturn wrap%s((*C.%s)(unsafe.Pointer(c))), nil\n", goType, cType))
	buf.WriteString("}\n\n")
}
