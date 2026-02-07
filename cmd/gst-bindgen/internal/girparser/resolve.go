package girparser

import (
	"fmt"
	"strings"
)

// TypeKind classifies how a type should be wrapped in Go.
type TypeKind int

const (
	TypeKindUnknown    TypeKind = iota
	TypeKindGObject             // Derives from GObject/GInitiallyUnowned
	TypeKindMiniObject          // Derives from GstMiniObject
	TypeKindRecord              // Plain C struct
	TypeKindEnum                // Enumeration
	TypeKindBitfield            // Flags/bitfield
	TypeKindCallback            // Function pointer type
	TypeKindInterface           // GObject interface
	TypeKindAlias               // Type alias
)

func (k TypeKind) String() string {
	switch k {
	case TypeKindGObject:
		return "GObject"
	case TypeKindMiniObject:
		return "MiniObject"
	case TypeKindRecord:
		return "Record"
	case TypeKindEnum:
		return "Enum"
	case TypeKindBitfield:
		return "Bitfield"
	case TypeKindCallback:
		return "Callback"
	case TypeKindInterface:
		return "Interface"
	case TypeKindAlias:
		return "Alias"
	default:
		return "Unknown"
	}
}

// ResolvedType holds resolved information about a type.
type ResolvedType struct {
	Kind        TypeKind
	Name        string // GIR name (e.g., "Element")
	CType       string // C type (e.g., "GstElement")
	Namespace   string // Owning namespace (e.g., "Gst")
	ParentChain []string // Fully qualified parent names (e.g., ["Gst.Bin", "Gst.Element", "Gst.Object"])
}

// TypeRegistry holds all parsed GIR data and resolved type information.
type TypeRegistry struct {
	repositories []*Repository
	namespaces   map[string]*Namespace // key: "Name-Version" (e.g., "Gst-1.0")
	types        map[string]*ResolvedType // key: "Namespace.TypeName" (e.g., "Gst.Element")
}

// NewTypeRegistry creates an empty type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		namespaces: make(map[string]*Namespace),
		types:      make(map[string]*ResolvedType),
	}
}

// AddRepository adds a parsed GIR repository to the registry.
func (r *TypeRegistry) AddRepository(repo *Repository) {
	r.repositories = append(r.repositories, repo)
	ns := &repo.Namespace
	key := ns.Name + "-" + ns.Version
	r.namespaces[key] = ns
}

// Namespace returns the namespace for a given name and version.
func (r *TypeRegistry) Namespace(name, version string) *Namespace {
	return r.namespaces[name+"-"+version]
}

// Resolve processes all registered namespaces and resolves type information.
func (r *TypeRegistry) Resolve() error {
	// First pass: register all types.
	for _, repo := range r.repositories {
		ns := &repo.Namespace
		for i := range ns.Classes {
			r.registerClass(ns.Name, &ns.Classes[i])
		}
		for i := range ns.Records {
			r.registerRecord(ns.Name, &ns.Records[i])
		}
		for i := range ns.Interfaces {
			r.registerInterface(ns.Name, &ns.Interfaces[i])
		}
		for i := range ns.Enumerations {
			r.registerEnum(ns.Name, &ns.Enumerations[i])
		}
		for i := range ns.Bitfields {
			r.registerBitfield(ns.Name, &ns.Bitfields[i])
		}
		for i := range ns.Callbacks {
			r.registerCallback(ns.Name, &ns.Callbacks[i])
		}
		for i := range ns.Aliases {
			r.registerAlias(ns.Name, &ns.Aliases[i])
		}
	}

	// Second pass: classify GObject vs MiniObject by walking parent chains.
	for _, rt := range r.types {
		if rt.Kind == TypeKindUnknown {
			r.classifyType(rt)
		}
	}

	return nil
}

// LookupType returns the resolved type for a fully qualified name (e.g., "Gst.Element").
func (r *TypeRegistry) LookupType(qualifiedName string) *ResolvedType {
	return r.types[qualifiedName]
}

// TypesInNamespace returns all resolved types belonging to a namespace.
func (r *TypeRegistry) TypesInNamespace(nsName string) []*ResolvedType {
	prefix := nsName + "."
	var result []*ResolvedType
	for key, rt := range r.types {
		if strings.HasPrefix(key, prefix) {
			result = append(result, rt)
		}
	}
	return result
}

func (r *TypeRegistry) registerClass(nsName string, cls *Class) {
	qname := nsName + "." + cls.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindUnknown, // Will be classified in second pass.
		Name:      cls.Name,
		CType:     cls.CType,
		Namespace: nsName,
	}
}

func (r *TypeRegistry) registerRecord(nsName string, rec *Record) {
	qname := nsName + "." + rec.Name
	kind := TypeKindRecord

	// Detect mini-objects: records whose first field is "mini_object" of type "MiniObject".
	if isMiniObjectRecord(rec) {
		kind = TypeKindMiniObject
	}

	r.types[qname] = &ResolvedType{
		Kind:      kind,
		Name:      rec.Name,
		CType:     rec.CType,
		Namespace: nsName,
	}
}

// isMiniObjectRecord checks if a record represents a GStreamer mini-object.
// Detection strategy:
//  1. Check for a field named "mini_object" with type "MiniObject" (non-opaque records).
//  2. For opaque records (no fields), check if the record has a glib:get-type and
//     has both "ref" and "unref" methods (characteristic of mini-objects).
func isMiniObjectRecord(rec *Record) bool {
	// Strategy 1: explicit mini_object field.
	for _, field := range rec.Fields {
		if field.Name == "mini_object" && field.Type != nil && field.Type.Name == "MiniObject" {
			return true
		}
	}

	// Strategy 2: opaque record with glib:get-type + ref/unref methods.
	if rec.GLibGetType != "" && len(rec.Fields) == 0 {
		hasRef := false
		hasUnref := false
		for _, m := range rec.Methods {
			if m.Name == "ref" {
				hasRef = true
			}
			if m.Name == "unref" {
				hasUnref = true
			}
		}
		if hasRef && hasUnref {
			return true
		}
	}

	return false
}

func (r *TypeRegistry) registerInterface(nsName string, iface *Interface) {
	qname := nsName + "." + iface.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindInterface,
		Name:      iface.Name,
		CType:     iface.CType,
		Namespace: nsName,
	}
}

func (r *TypeRegistry) registerEnum(nsName string, enum *Enumeration) {
	qname := nsName + "." + enum.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindEnum,
		Name:      enum.Name,
		CType:     enum.CType,
		Namespace: nsName,
	}
}

func (r *TypeRegistry) registerBitfield(nsName string, bf *Bitfield) {
	qname := nsName + "." + bf.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindBitfield,
		Name:      bf.Name,
		CType:     bf.CType,
		Namespace: nsName,
	}
}

func (r *TypeRegistry) registerCallback(nsName string, cb *Callback) {
	qname := nsName + "." + cb.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindCallback,
		Name:      cb.Name,
		CType:     cb.CType,
		Namespace: nsName,
	}
}

func (r *TypeRegistry) registerAlias(nsName string, alias *Alias) {
	qname := nsName + "." + alias.Name
	r.types[qname] = &ResolvedType{
		Kind:      TypeKindAlias,
		Name:      alias.Name,
		CType:     alias.CType,
		Namespace: nsName,
	}
}

// classifyType determines whether a class is GObject-derived or MiniObject-derived
// by walking its parent chain.
func (r *TypeRegistry) classifyType(rt *ResolvedType) TypeKind {
	if rt.Kind != TypeKindUnknown {
		return rt.Kind
	}

	// Find the class in its namespace to get the parent.
	parent := r.findClassParent(rt.Namespace, rt.Name)
	if parent == "" {
		// No parent — treat as GObject by default (most GStreamer classes are).
		rt.Kind = TypeKindGObject
		return rt.Kind
	}

	// Resolve the qualified parent name.
	qualParent := r.qualifyName(rt.Namespace, parent)

	// Check well-known base types.
	switch qualParent {
	case "GObject.Object", "GObject.InitiallyUnowned":
		rt.Kind = TypeKindGObject
		rt.ParentChain = []string{qualParent}
		return rt.Kind
	case "Gst.MiniObject":
		rt.Kind = TypeKindMiniObject
		rt.ParentChain = []string{qualParent}
		return rt.Kind
	}

	// Recursively classify the parent.
	parentRT := r.types[qualParent]
	if parentRT == nil {
		// Unknown parent — default to GObject.
		rt.Kind = TypeKindGObject
		return rt.Kind
	}

	parentKind := r.classifyType(parentRT)
	rt.Kind = parentKind
	rt.ParentChain = append([]string{qualParent}, parentRT.ParentChain...)
	return rt.Kind
}

// findClassParent looks up a class by name within a namespace and returns its parent attribute.
func (r *TypeRegistry) findClassParent(nsName, className string) string {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for _, cls := range repo.Namespace.Classes {
			if cls.Name == className {
				return cls.Parent
			}
		}
	}
	return ""
}

// QualifyName resolves a potentially unqualified GIR type name to a fully qualified one.
// If the name already contains a dot, it's already qualified.
// Otherwise, it's assumed to be in the given default namespace.
func (r *TypeRegistry) QualifyName(defaultNS, name string) string {
	return r.qualifyName(defaultNS, name)
}

// qualifyName resolves a potentially unqualified GIR type name to a fully qualified one.
func (r *TypeRegistry) qualifyName(defaultNS, name string) string {
	if strings.Contains(name, ".") {
		return name
	}
	// Try the default namespace first.
	qname := defaultNS + "." + name
	if _, ok := r.types[qname]; ok {
		return qname
	}
	// Search all namespaces.
	for _, repo := range r.repositories {
		candidate := repo.Namespace.Name + "." + name
		if _, ok := r.types[candidate]; ok {
			return candidate
		}
	}
	// Fall back to default namespace.
	return qname
}

// FindClass looks up a class by fully qualified name (e.g., "Gst.Element").
func (r *TypeRegistry) FindClass(qualifiedName string) *Class {
	parts := strings.SplitN(qualifiedName, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	nsName, className := parts[0], parts[1]
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Classes {
			if repo.Namespace.Classes[i].Name == className {
				return &repo.Namespace.Classes[i]
			}
		}
	}
	return nil
}

// FindEnumFunction looks for a function on an enumeration (e.g., a _get_name function).
func (r *TypeRegistry) FindEnumFunction(nsName, enumName, funcName string) *Function {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for _, enum := range repo.Namespace.Enumerations {
			if enum.Name == enumName {
				for i := range enum.Functions {
					if enum.Functions[i].Name == funcName {
						return &enum.Functions[i]
					}
				}
			}
		}
		for _, bf := range repo.Namespace.Bitfields {
			if bf.Name == enumName {
				for i := range bf.Functions {
					if bf.Functions[i].Name == funcName {
						return &bf.Functions[i]
					}
				}
			}
		}
	}
	return nil
}

// AllNamespaces returns all namespace keys.
func (r *TypeRegistry) AllNamespaces() []string {
	keys := make([]string, 0, len(r.namespaces))
	for k := range r.namespaces {
		keys = append(keys, k)
	}
	return keys
}

// FindEnumByName finds an enumeration by namespace and name.
func (r *TypeRegistry) FindEnumByName(nsName, name string) (*Enumeration, error) {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Enumerations {
			if repo.Namespace.Enumerations[i].Name == name {
				return &repo.Namespace.Enumerations[i], nil
			}
		}
	}
	return nil, fmt.Errorf("enumeration %s.%s not found", nsName, name)
}

// FindRecord looks up a record by fully qualified name (e.g., "Gst.Buffer").
func (r *TypeRegistry) FindRecord(qualifiedName string) *Record {
	parts := strings.SplitN(qualifiedName, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	nsName, recName := parts[0], parts[1]
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Records {
			if repo.Namespace.Records[i].Name == recName {
				return &repo.Namespace.Records[i]
			}
		}
	}
	return nil
}

// FindRecordByName finds a record by namespace and name.
func (r *TypeRegistry) FindRecordByName(nsName, name string) *Record {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Records {
			if repo.Namespace.Records[i].Name == name {
				return &repo.Namespace.Records[i]
			}
		}
	}
	return nil
}

// FindInterface looks up an interface by fully qualified name.
func (r *TypeRegistry) FindInterface(qualifiedName string) *Interface {
	parts := strings.SplitN(qualifiedName, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	nsName, ifaceName := parts[0], parts[1]
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Interfaces {
			if repo.Namespace.Interfaces[i].Name == ifaceName {
				return &repo.Namespace.Interfaces[i]
			}
		}
	}
	return nil
}

// FindClassByName finds a class by namespace and name.
func (r *TypeRegistry) FindClassByName(nsName, name string) *Class {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Classes {
			if repo.Namespace.Classes[i].Name == name {
				return &repo.Namespace.Classes[i]
			}
		}
	}
	return nil
}

// FindBitfieldByName finds a bitfield by namespace and name.
func (r *TypeRegistry) FindBitfieldByName(nsName, name string) (*Bitfield, error) {
	for _, repo := range r.repositories {
		if repo.Namespace.Name != nsName {
			continue
		}
		for i := range repo.Namespace.Bitfields {
			if repo.Namespace.Bitfields[i].Name == name {
				return &repo.Namespace.Bitfields[i], nil
			}
		}
	}
	return nil, fmt.Errorf("bitfield %s.%s not found", nsName, name)
}
