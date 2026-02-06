package overrides

// PackageOverrides defines overrides for a single Go package's code generation.
type PackageOverrides struct {
	Package   string `yaml:"package"`
	Namespace string `yaml:"namespace"`

	// SkipTypes lists type names that should not be generated (remain hand-written).
	SkipTypes []string `yaml:"skip_types"`

	// SkipMethods maps type names to lists of method names that should not be generated.
	SkipMethods map[string][]string `yaml:"skip_methods"`

	// SkipFunctions lists standalone function names that should not be generated.
	SkipFunctions []string `yaml:"skip_functions"`

	// CastMacros overrides the default GST_TYPE() cast macro for specific C types.
	// Key: C type name (e.g., "GstDevice"), Value: macro name (e.g., "GST_DEVICE_CAST").
	CastMacros map[string]string `yaml:"cast_macros"`

	// MiniObjectExtraFields adds extra Go struct fields to mini-object types.
	MiniObjectExtraFields map[string][]ExtraField `yaml:"miniobject_extra_fields"`

	// AnnotationFixes corrects broken GIR annotations.
	AnnotationFixes map[string]AnnotationFix `yaml:"annotation_fixes"`

	// Appends contains verbatim Go code to append to specific generated files.
	// Key: target file identifier (e.g., "constants"), Value: Go source code.
	Appends map[string]string `yaml:"appends"`

	// Renames overrides auto-generated Go names.
	Renames *RenameOverrides `yaml:"renames"`
}

// ExtraField defines an additional struct field for a generated type.
type ExtraField struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// AnnotationFix corrects GIR annotations for a specific function.
type AnnotationFix struct {
	Params map[string]ParamFix `yaml:"params"`
	Return *ReturnFix          `yaml:"return"`
}

// ParamFix corrects a parameter's annotations.
type ParamFix struct {
	Direction         string `yaml:"direction"`
	TransferOwnership string `yaml:"transfer_ownership"`
	Nullable          *bool  `yaml:"nullable"`
}

// ReturnFix corrects a return value's annotations.
type ReturnFix struct {
	TransferOwnership string `yaml:"transfer_ownership"`
	Nullable          *bool  `yaml:"nullable"`
}

// RenameOverrides allows overriding auto-generated names.
type RenameOverrides struct {
	Types   map[string]string `yaml:"types"`
	Methods map[string]string `yaml:"methods"`
}

// IsTypeSkipped returns true if the given type name should not be generated.
func (o *PackageOverrides) IsTypeSkipped(typeName string) bool {
	for _, name := range o.SkipTypes {
		if name == typeName {
			return true
		}
	}
	return false
}

// IsMethodSkipped returns true if the given method on a type should not be generated.
func (o *PackageOverrides) IsMethodSkipped(typeName, methodName string) bool {
	methods, ok := o.SkipMethods[typeName]
	if !ok {
		return false
	}
	for _, name := range methods {
		if name == methodName {
			return true
		}
	}
	return false
}

// IsFunctionSkipped returns true if the given function should not be generated.
func (o *PackageOverrides) IsFunctionSkipped(funcName string) bool {
	for _, name := range o.SkipFunctions {
		if name == funcName {
			return true
		}
	}
	return false
}

// GetCastMacro returns the C cast macro for a given C type, or empty string for default.
func (o *PackageOverrides) GetCastMacro(cType string) string {
	if macro, ok := o.CastMacros[cType]; ok {
		return macro
	}
	return ""
}
