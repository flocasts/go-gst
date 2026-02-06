package typeconv

// GoType represents a Go type with its import path and conversion logic.
type GoType struct {
	Name       string // Go type name (e.g., "string", "bool", "*Element")
	Import     string // Import path if needed (e.g., "unsafe")
	IsPointer  bool   // Whether this is a pointer type
	IsEnum     bool   // Whether this is an enum type
	IsPrimitive bool  // Whether this is a primitive type (int, bool, string, etc.)
}

// PrimitiveTypeMap maps GIR primitive type names to Go types.
var PrimitiveTypeMap = map[string]GoType{
	"utf8":     {Name: "string", IsPrimitive: true},
	"filename": {Name: "string", IsPrimitive: true},
	"gboolean": {Name: "bool", IsPrimitive: true},
	"gint":     {Name: "int", IsPrimitive: true},
	"guint":    {Name: "uint", IsPrimitive: true},
	"gint8":    {Name: "int8", IsPrimitive: true},
	"guint8":   {Name: "uint8", IsPrimitive: true},
	"gint16":   {Name: "int16", IsPrimitive: true},
	"guint16":  {Name: "uint16", IsPrimitive: true},
	"gint32":   {Name: "int32", IsPrimitive: true},
	"guint32":  {Name: "uint32", IsPrimitive: true},
	"gint64":   {Name: "int64", IsPrimitive: true},
	"guint64":  {Name: "uint64", IsPrimitive: true},
	"glong":    {Name: "int", IsPrimitive: true},
	"gulong":   {Name: "uint", IsPrimitive: true},
	"gfloat":   {Name: "float32", IsPrimitive: true},
	"gdouble":  {Name: "float64", IsPrimitive: true},
	"gsize":    {Name: "uint", IsPrimitive: true},
	"gssize":   {Name: "int", IsPrimitive: true},
	"gpointer": {Name: "unsafe.Pointer", Import: "unsafe", IsPrimitive: true},
	"none":     {Name: "", IsPrimitive: true}, // void
	"gchar":    {Name: "byte", IsPrimitive: true},
	"guchar":   {Name: "byte", IsPrimitive: true},
}

// CTypeToGoConversion describes how to convert between C and Go for a parameter.
type CTypeToGoConversion struct {
	// ToC is Go code template to convert a Go value to C.
	// Uses {{.Name}} for the Go variable name.
	ToC string

	// ToGo is Go code template to convert a C value to Go.
	// Uses {{.Name}} for the C variable name.
	ToGo string

	// NeedsDefer is true if the conversion requires a defer (e.g., C.free for strings).
	NeedsDefer bool

	// DeferCode is the defer statement if NeedsDefer is true.
	DeferCode string
}

// PrimitiveConversions maps GIR type names to conversion templates.
var PrimitiveConversions = map[string]CTypeToGoConversion{
	"utf8": {
		ToC:        "C.CString({{.Name}})",
		ToGo:       "C.GoString({{.Name}})",
		NeedsDefer: true,
		DeferCode:  "C.free(unsafe.Pointer({{.CName}}))",
	},
	"gboolean": {
		ToC:  "gboolean({{.Name}})",
		ToGo: "gobool({{.Name}})",
	},
	"gint": {
		ToC:  "C.gint({{.Name}})",
		ToGo: "int({{.Name}})",
	},
	"guint": {
		ToC:  "C.guint({{.Name}})",
		ToGo: "uint({{.Name}})",
	},
	"gint64": {
		ToC:  "C.gint64({{.Name}})",
		ToGo: "int64({{.Name}})",
	},
	"guint64": {
		ToC:  "C.guint64({{.Name}})",
		ToGo: "uint64({{.Name}})",
	},
	"gfloat": {
		ToC:  "C.gfloat({{.Name}})",
		ToGo: "float32({{.Name}})",
	},
	"gdouble": {
		ToC:  "C.gdouble({{.Name}})",
		ToGo: "float64({{.Name}})",
	},
	"gsize": {
		ToC:  "C.gsize({{.Name}})",
		ToGo: "uint({{.Name}})",
	},
	"gssize": {
		ToC:  "C.gssize({{.Name}})",
		ToGo: "int({{.Name}})",
	},
}

// TransferOwnership describes how a return value's ownership is transferred.
type TransferOwnership string

const (
	TransferNone      TransferOwnership = "none"
	TransferFull      TransferOwnership = "full"
	TransferContainer TransferOwnership = "container"
)

// ReturnWrapping determines how to wrap a C return value for a given type and transfer mode.
func ReturnWrapping(goType string, transfer TransferOwnership, isMiniObject bool) string {
	switch transfer {
	case TransferFull:
		if isMiniObject {
			return "FromGst" + goType + "UnsafeFull"
		}
		return "FromGst" + goType + "UnsafeFull"
	case TransferNone:
		if isMiniObject {
			return "FromGst" + goType + "UnsafeNone"
		}
		return "FromGst" + goType + "UnsafeNone"
	default:
		return "FromGst" + goType + "UnsafeFull"
	}
}
