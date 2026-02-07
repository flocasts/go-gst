package apidiff

// PackageAPI captures the exported API surface of a Go package
// by extracting declarations from generated _gen_*.go files.
type PackageAPI struct {
	Package   string
	Types     map[string]TypeDef
	Functions map[string]FuncSig
	Methods   map[string]FuncSig // key: "ReceiverType.MethodName"
	Constants map[string]ConstDef
}

// TypeDef represents an exported type definition.
type TypeDef struct {
	Name       string
	Underlying string  // "struct", "int", "uint", func signature, etc.
	Fields     []Field // exported struct fields (empty for non-structs)
}

// Field represents an exported struct field.
type Field struct {
	Name string
	Type string
}

// FuncSig represents an exported function or method signature.
type FuncSig struct {
	Name     string
	Receiver string // empty for package-level functions
	Params   []Param
	Returns  []string
}

// Param represents a function parameter.
type Param struct {
	Name string
	Type string
}

// ConstDef represents an exported constant.
type ConstDef struct {
	Name  string
	Type  string
	Value string // raw expression string (e.g. "C.GST_STATE_PLAYING")
}

// ChangeKind classifies an API change.
type ChangeKind int

const (
	TypeAdded ChangeKind = iota
	TypeRemoved
	TypeChanged
	FuncAdded
	FuncRemoved
	FuncSignatureChanged
	MethodAdded
	MethodRemoved
	MethodSignatureChanged
	ConstAdded
	ConstRemoved
	ConstTypeChanged
)

// Change represents a single API change between two package versions.
type Change struct {
	Kind       ChangeKind
	Symbol     string // fully qualified symbol name
	Old        string // human-readable old definition (empty for additions)
	New        string // human-readable new definition (empty for removals)
	IsBreaking bool
}

// DiffReport contains all API changes for a single package.
type DiffReport struct {
	Package string
	Changes []Change
}

// HasBreaking returns true if any change in the report is breaking.
func (r *DiffReport) HasBreaking() bool {
	for _, c := range r.Changes {
		if c.IsBreaking {
			return true
		}
	}
	return false
}

// BreakingCount returns the number of breaking changes.
func (r *DiffReport) BreakingCount() int {
	n := 0
	for _, c := range r.Changes {
		if c.IsBreaking {
			n++
		}
	}
	return n
}

// NonBreakingCount returns the number of non-breaking changes.
func (r *DiffReport) NonBreakingCount() int {
	n := 0
	for _, c := range r.Changes {
		if !c.IsBreaking {
			n++
		}
	}
	return n
}

// NewPackageAPI creates an empty PackageAPI for the given package name.
func NewPackageAPI(pkg string) *PackageAPI {
	return &PackageAPI{
		Package:   pkg,
		Types:     make(map[string]TypeDef),
		Functions: make(map[string]FuncSig),
		Methods:   make(map[string]FuncSig),
		Constants: make(map[string]ConstDef),
	}
}
