package girparser

import "encoding/xml"

// Repository is the root element of a GIR file.
type Repository struct {
	XMLName    xml.Name   `xml:"repository"`
	Version    string     `xml:"version,attr"`
	Includes   []Include  `xml:"include"`
	Packages   []Package  `xml:"package"`
	Namespace  Namespace  `xml:"namespace"`
}

// Include references another GIR namespace.
type Include struct {
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr"`
}

// Package identifies a pkg-config package.
type Package struct {
	Name string `xml:"name,attr"`
}

// Namespace contains all type definitions for a library.
type Namespace struct {
	Name              string          `xml:"name,attr"`
	Version           string          `xml:"version,attr"`
	CIdentifierPrefix string          `xml:"http://www.gtk.org/introspection/c/1.0 identifier-prefixes,attr"`
	CSymbolPrefix     string          `xml:"http://www.gtk.org/introspection/c/1.0 symbol-prefixes,attr"`
	SharedLibrary     string          `xml:"shared-library,attr"`
	Classes           []Class         `xml:"class"`
	Records           []Record        `xml:"record"`
	Interfaces        []Interface     `xml:"interface"`
	Enumerations      []Enumeration   `xml:"enumeration"`
	Bitfields         []Bitfield      `xml:"bitfield"`
	Functions         []Function      `xml:"function"`
	Callbacks         []Callback      `xml:"callback"`
	Constants         []Constant      `xml:"constant"`
	Aliases           []Alias         `xml:"alias"`
}

// Class represents a GObject class.
type Class struct {
	Name           string          `xml:"name,attr"`
	CType          string          `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	CSymbolPrefix  string          `xml:"http://www.gtk.org/introspection/c/1.0 symbol-prefix,attr"`
	Parent         string          `xml:"parent,attr"`
	GLibTypeName   string          `xml:"http://www.gtk.org/introspection/glib/1.0 type-name,attr"`
	GLibGetType    string          `xml:"http://www.gtk.org/introspection/glib/1.0 get-type,attr"`
	Abstract       bool            `xml:"abstract,attr"`
	Doc            *Doc            `xml:"doc"`
	Constructors   []Function      `xml:"constructor"`
	Methods        []Method        `xml:"method"`
	Functions      []Function      `xml:"function"`
	VirtualMethods []VirtualMethod `xml:"virtual-method"`
	Properties     []Property      `xml:"property"`
	Signals        []Signal        `xml:"http://www.gtk.org/introspection/glib/1.0 signal"`
	Implements     []Implement     `xml:"implements"`
	Fields         []Field         `xml:"field"`
}

// Record represents a C struct (non-GObject).
type Record struct {
	Name          string     `xml:"name,attr"`
	CType         string     `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	CSymbolPrefix string     `xml:"http://www.gtk.org/introspection/c/1.0 symbol-prefix,attr"`
	GLibTypeName  string     `xml:"http://www.gtk.org/introspection/glib/1.0 type-name,attr"`
	GLibGetType   string     `xml:"http://www.gtk.org/introspection/glib/1.0 get-type,attr"`
	Disguised     bool       `xml:"disguised,attr"`
	Foreign       bool       `xml:"foreign,attr"`
	Doc           *Doc       `xml:"doc"`
	Constructors  []Function `xml:"constructor"`
	Methods       []Method   `xml:"method"`
	Functions     []Function `xml:"function"`
	Fields        []Field    `xml:"field"`
}

// Interface represents a GObject interface.
type Interface struct {
	Name           string          `xml:"name,attr"`
	CType          string          `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	CSymbolPrefix  string          `xml:"http://www.gtk.org/introspection/c/1.0 symbol-prefix,attr"`
	GLibTypeName   string          `xml:"http://www.gtk.org/introspection/glib/1.0 type-name,attr"`
	GLibGetType    string          `xml:"http://www.gtk.org/introspection/glib/1.0 get-type,attr"`
	Doc            *Doc            `xml:"doc"`
	Prerequisites  []Prerequisite  `xml:"prerequisite"`
	Methods        []Method        `xml:"method"`
	Functions      []Function      `xml:"function"`
	VirtualMethods []VirtualMethod `xml:"virtual-method"`
	Properties     []Property      `xml:"property"`
	Signals        []Signal        `xml:"http://www.gtk.org/introspection/glib/1.0 signal"`
}

// Enumeration represents a C enum.
type Enumeration struct {
	Name        string   `xml:"name,attr"`
	CType       string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	GLibTypeName string  `xml:"http://www.gtk.org/introspection/glib/1.0 type-name,attr"`
	GLibGetType string   `xml:"http://www.gtk.org/introspection/glib/1.0 get-type,attr"`
	Doc         *Doc     `xml:"doc"`
	Members     []Member `xml:"member"`
	Functions   []Function `xml:"function"`
}

// Bitfield represents C flags (bitwise OR-able enum).
type Bitfield struct {
	Name         string   `xml:"name,attr"`
	CType        string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	GLibTypeName string   `xml:"http://www.gtk.org/introspection/glib/1.0 type-name,attr"`
	GLibGetType  string   `xml:"http://www.gtk.org/introspection/glib/1.0 get-type,attr"`
	Doc          *Doc     `xml:"doc"`
	Members      []Member `xml:"member"`
	Functions    []Function `xml:"function"`
}

// Member is a single value in an enum or bitfield.
type Member struct {
	Name        string `xml:"name,attr"`
	Value       string `xml:"value,attr"`
	CIdentifier string `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	GLibNick    string `xml:"http://www.gtk.org/introspection/glib/1.0 nick,attr"`
	Doc         *Doc   `xml:"doc"`
}

// Function represents a standalone function or constructor.
type Function struct {
	Name          string      `xml:"name,attr"`
	CIdentifier   string      `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Version       string      `xml:"version,attr"`
	Deprecated    string      `xml:"deprecated,attr"`
	DeprecatedVer string      `xml:"deprecated-version,attr"`
	Introspectable string     `xml:"introspectable,attr"`
	Throws        bool        `xml:"throws,attr"`
	Doc           *Doc        `xml:"doc"`
	ReturnValue   *ReturnValue `xml:"return-value"`
	Parameters    *Parameters  `xml:"parameters"`
}

// Method represents an instance method on a class, record, or interface.
type Method struct {
	Name          string      `xml:"name,attr"`
	CIdentifier   string      `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Version       string      `xml:"version,attr"`
	Deprecated    string      `xml:"deprecated,attr"`
	DeprecatedVer string      `xml:"deprecated-version,attr"`
	Introspectable string     `xml:"introspectable,attr"`
	Throws        bool        `xml:"throws,attr"`
	Doc           *Doc        `xml:"doc"`
	ReturnValue   *ReturnValue `xml:"return-value"`
	Parameters    *Parameters  `xml:"parameters"`
}

// VirtualMethod represents a virtual/overridable method.
type VirtualMethod struct {
	Name          string      `xml:"name,attr"`
	CIdentifier   string      `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Invoker       string      `xml:"invoker,attr"`
	Doc           *Doc        `xml:"doc"`
	ReturnValue   *ReturnValue `xml:"return-value"`
	Parameters    *Parameters  `xml:"parameters"`
}

// Callback represents a callback function pointer type.
type Callback struct {
	Name        string      `xml:"name,attr"`
	CType       string      `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Doc         *Doc        `xml:"doc"`
	ReturnValue *ReturnValue `xml:"return-value"`
	Parameters  *Parameters  `xml:"parameters"`
}

// Signal represents a GObject signal.
type Signal struct {
	Name        string      `xml:"name,attr"`
	When        string      `xml:"when,attr"`
	Detailed    bool        `xml:"detailed,attr"`
	Doc         *Doc        `xml:"doc"`
	ReturnValue *ReturnValue `xml:"return-value"`
	Parameters  *Parameters  `xml:"parameters"`
}

// Parameters is the container for parameters.
type Parameters struct {
	InstanceParam *Parameter  `xml:"instance-parameter"`
	Params        []Parameter `xml:"parameter"`
}

// Parameter describes a function/method parameter.
type Parameter struct {
	Name              string   `xml:"name,attr"`
	Direction         string   `xml:"direction,attr"`
	TransferOwnership string   `xml:"transfer-ownership,attr"`
	Nullable          bool     `xml:"nullable,attr"`
	Optional          bool     `xml:"optional,attr"`
	CallerAllocates   bool     `xml:"caller-allocates,attr"`
	Closure           string   `xml:"closure,attr"`
	Destroy           string   `xml:"destroy,attr"`
	Scope             string   `xml:"scope,attr"`
	Doc               *Doc     `xml:"doc"`
	Type              *TypeRef `xml:"type"`
	Array             *Array   `xml:"array"`
	Varargs           *struct{} `xml:"varargs"`
}

// ReturnValue describes the return value of a function/method.
type ReturnValue struct {
	TransferOwnership string   `xml:"transfer-ownership,attr"`
	Nullable          bool     `xml:"nullable,attr"`
	Doc               *Doc     `xml:"doc"`
	Type              *TypeRef `xml:"type"`
	Array             *Array   `xml:"array"`
}

// TypeRef references a type by name.
type TypeRef struct {
	Name  string `xml:"name,attr"`
	CType string `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
}

// Array describes an array type.
type Array struct {
	Name           string   `xml:"name,attr"`
	CType          string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Length         string   `xml:"length,attr"`
	ZeroTerminated string   `xml:"zero-terminated,attr"`
	FixedSize      string   `xml:"fixed-size,attr"`
	Type           *TypeRef `xml:"type"`
}

// Property describes a GObject property.
type Property struct {
	Name              string   `xml:"name,attr"`
	Writable          bool     `xml:"writable,attr"`
	Readable          bool     `xml:"readable,attr"`
	Construct         bool     `xml:"construct,attr"`
	ConstructOnly     bool     `xml:"construct-only,attr"`
	TransferOwnership string   `xml:"transfer-ownership,attr"`
	Version           string   `xml:"version,attr"`
	Deprecated        string   `xml:"deprecated,attr"`
	Doc               *Doc     `xml:"doc"`
	Type              *TypeRef `xml:"type"`
}

// Field describes a struct field.
type Field struct {
	Name     string   `xml:"name,attr"`
	Writable bool     `xml:"writable,attr"`
	Readable bool     `xml:"readable,attr"`
	Private  bool     `xml:"private,attr"`
	Bits     int      `xml:"bits,attr"`
	Type     *TypeRef `xml:"type"`
	Array    *Array   `xml:"array"`
	Callback *Callback `xml:"callback"`
}

// Implement references an interface that a class implements.
type Implement struct {
	Name string `xml:"name,attr"`
}

// Prerequisite references a prerequisite for an interface.
type Prerequisite struct {
	Name string `xml:"name,attr"`
}

// Constant defines a constant value.
type Constant struct {
	Name        string   `xml:"name,attr"`
	Value       string   `xml:"value,attr"`
	CType       string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	CIdentifier string   `xml:"http://www.gtk.org/introspection/c/1.0 identifier,attr"`
	Doc         *Doc     `xml:"doc"`
	Type        *TypeRef `xml:"type"`
}

// Alias defines a type alias.
type Alias struct {
	Name  string   `xml:"name,attr"`
	CType string   `xml:"http://www.gtk.org/introspection/c/1.0 type,attr"`
	Doc   *Doc     `xml:"doc"`
	Type  *TypeRef `xml:"type"`
}

// Doc holds documentation text.
type Doc struct {
	Text string `xml:",chardata"`
}
