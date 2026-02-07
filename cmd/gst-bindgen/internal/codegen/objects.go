package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-gst/go-gst/cmd/gst-bindgen/internal/girparser"
)

// generateTypes generates the _gen_types.go file with GObject type scaffolding.
func generateTypes(ctx *PackageContext) error {
	ns := ctx.Namespace

	// Collect GObject-derived classes that should be generated.
	var classes []girparser.Class
	for _, cls := range ns.Classes {
		if ctx.Overrides.IsTypeSkipped(cls.Name) {
			continue
		}
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		if rt == nil || rt.Kind != girparser.TypeKindGObject {
			continue
		}
		classes = append(classes, cls)
	}

	if len(classes) == 0 {
		return nil
	}

	sort.Slice(classes, func(i, j int) bool {
		return classes[i].Name < classes[j].Name
	})

	var buf strings.Builder

	writeGeneratedHeader(&buf, ctx)
	buf.WriteString("package ")
	buf.WriteString(goPackageName(ctx.Package.GoPackage))
	buf.WriteString("\n\n")

	// Imports.
	buf.WriteString("/*\n#include \"gst.go.h\"\n*/\nimport \"C\"\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"unsafe\"\n\n")
	buf.WriteString("\t\"github.com/go-gst/go-glib/glib\"\n")
	buf.WriteString(")\n\n")

	for _, cls := range classes {
		rt := ctx.Registry.LookupType(ctx.Package.Namespace + "." + cls.Name)
		writeGObjectType(&buf, ctx, cls, rt)
	}

	outPath := ctx.genFilePath("_gen_types.go")
	return writeFileIfChanged(outPath, buf.String())
}

// writeGObjectType writes the struct, FromUnsafe*, Instance(), and wrap function for a GObject class.
func writeGObjectType(buf *strings.Builder, ctx *PackageContext, cls girparser.Class, rt *girparser.ResolvedType) {
	goType := GIRNameToGoType(cls.Name)
	receiver := GoTypeToReceiverName(goType)

	// Determine the parent Go type and how to embed it.
	parentEmbed := resolveParentEmbed(ctx, cls, rt)

	// Doc comment.
	if cls.Doc != nil && cls.Doc.Text != "" {
		writeDocComment(buf, goType, cls.Doc.Text)
	}

	// Struct definition.
	if parentEmbed.fieldName == "" {
		// Direct embed (anonymous): type Element struct{ *Object }
		buf.WriteString(fmt.Sprintf("type %s struct{ %s }\n\n", goType, parentEmbed.goTypeExpr))
	} else {
		// Named field: type Pipeline struct{ Bin *Bin }
		buf.WriteString(fmt.Sprintf("type %s struct{ %s %s }\n\n", goType, parentEmbed.fieldName, parentEmbed.goTypeExpr))
	}

	// wrapXxx function.
	writeWrapGObject(buf, ctx, cls, goType, parentEmbed)

	// FromGstXxxUnsafeNone.
	buf.WriteString(fmt.Sprintf("// FromGst%sUnsafeNone wraps the pointer with transfer-none semantics.\n", goType))
	buf.WriteString(fmt.Sprintf("func FromGst%sUnsafeNone(ptr unsafe.Pointer) *%s {\n", goType, goType))
	buf.WriteString("\tif ptr == nil {\n\t\treturn nil\n\t}\n")
	buf.WriteString(fmt.Sprintf("\treturn wrap%s(glib.TransferNone(ptr))\n", goType))
	buf.WriteString("}\n\n")

	// FromGstXxxUnsafeFull.
	buf.WriteString(fmt.Sprintf("// FromGst%sUnsafeFull wraps the pointer with transfer-full semantics.\n", goType))
	buf.WriteString(fmt.Sprintf("func FromGst%sUnsafeFull(ptr unsafe.Pointer) *%s {\n", goType, goType))
	buf.WriteString("\tif ptr == nil {\n\t\treturn nil\n\t}\n")
	buf.WriteString(fmt.Sprintf("\treturn wrap%s(glib.TransferFull(ptr))\n", goType))
	buf.WriteString("}\n\n")

	// Instance() method.
	cType := cls.CType
	if cType == "" {
		cType = "Gst" + cls.Name
	}
	buf.WriteString(fmt.Sprintf("// Instance returns the native C %s pointer.\n", cType))
	buf.WriteString(fmt.Sprintf("func (%s *%s) Instance() *C.%s { return C.toGst%s(%s.Unsafe()) }\n\n",
		receiver, goType, cType, cls.Name, receiver))
}

// parentEmbedInfo describes how a GObject type embeds its parent.
type parentEmbedInfo struct {
	fieldName  string // empty for anonymous embed
	goTypeExpr string // e.g., "*Object" or "*gst.Element"
	wrapExpr   string // e.g., "wrapObject(obj)" or "wrapElement(obj)"
}

// resolveParentEmbed determines the Go embedding for a GObject class based on its parent chain.
func resolveParentEmbed(ctx *PackageContext, cls girparser.Class, rt *girparser.ResolvedType) parentEmbedInfo {
	if len(rt.ParentChain) == 0 || cls.Parent == "" {
		// Root type — embed glib.InitiallyUnowned.
		return parentEmbedInfo{
			goTypeExpr: "*glib.InitiallyUnowned",
			wrapExpr:   "&glib.InitiallyUnowned{Object: obj}",
		}
	}

	parentQual := rt.ParentChain[0] // e.g., "Gst.Element"
	parts := strings.SplitN(parentQual, ".", 2)
	if len(parts) != 2 {
		return parentEmbedInfo{
			goTypeExpr: "*glib.InitiallyUnowned",
			wrapExpr:   "&glib.InitiallyUnowned{Object: obj}",
		}
	}

	parentNS, parentName := parts[0], parts[1]
	parentGoType := GIRNameToGoType(parentName)

	// Check for well-known terminal parents.
	switch parentQual {
	case "GObject.Object", "GObject.InitiallyUnowned":
		return parentEmbedInfo{
			goTypeExpr: "*glib.InitiallyUnowned",
			wrapExpr:   "&glib.InitiallyUnowned{Object: obj}",
		}
	}

	// If the parent is in a different namespace, we need a qualified import.
	if parentNS != ctx.Package.Namespace {
		pkgAlias := goPackageName(namespaceToPkgPath(parentNS))
		return parentEmbedInfo{
			fieldName:  parentGoType,
			goTypeExpr: fmt.Sprintf("*%s.%s", pkgAlias, parentGoType),
			wrapExpr:   fmt.Sprintf("%s.FromGst%sUnsafeNone(unsafe.Pointer(obj.GObject))", pkgAlias, parentGoType),
		}
	}

	// Same namespace parent — check if it's Object (special base type).
	if parentName == "Object" {
		return parentEmbedInfo{
			goTypeExpr: "*Object",
			wrapExpr:   "wrapObject(obj)",
		}
	}

	// General case: named field with parent type.
	return parentEmbedInfo{
		fieldName:  parentGoType,
		goTypeExpr: "*" + parentGoType,
		wrapExpr:   fmt.Sprintf("wrap%s(obj)", parentGoType),
	}
}

// writeWrapGObject writes the wrapXxx function for a GObject type.
func writeWrapGObject(buf *strings.Builder, ctx *PackageContext, cls girparser.Class, goType string, parent parentEmbedInfo) {
	buf.WriteString(fmt.Sprintf("func wrap%s(obj *glib.Object) *%s {\n", goType, goType))
	if parent.fieldName == "" {
		// Anonymous embed.
		buf.WriteString(fmt.Sprintf("\treturn &%s{%s}\n", goType, parent.wrapExpr))
	} else {
		// Named field.
		buf.WriteString(fmt.Sprintf("\treturn &%s{%s: %s}\n", goType, parent.fieldName, parent.wrapExpr))
	}
	buf.WriteString("}\n\n")
}

// namespaceToPkgPath converts a GIR namespace to a Go package path.
func namespaceToPkgPath(ns string) string {
	for _, pkg := range DefaultPackages {
		if pkg.Namespace == ns {
			return pkg.GoPackage
		}
	}
	return strings.ToLower(ns)
}
