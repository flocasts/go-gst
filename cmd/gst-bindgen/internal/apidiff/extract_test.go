package apidiff

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFromTestData(t *testing.T) {
	// Write a small test fixture that mimics generated code patterns.
	dir := t.TempDir()

	src := `package testpkg

/*
#include "gst.go.h"
*/
import "C"

import "unsafe"

// Foo is a test type.
type Foo struct{ Bar *Bar }

type Bar struct{ *Baz }

type MyEnum int

const (
	EnumA MyEnum = 1
	EnumB MyEnum = 2
	EnumC MyEnum = 3
)

func NewFoo(name string) *Foo {
	return nil
}

func (f *Foo) GetBar() *Bar {
	return f.Bar
}

func (f *Foo) SetName(name string, flag bool) {
}

func (f *Foo) Instance() *C.GstFoo { return nil }

// unexported — should not appear
func helper() {}
func (f *Foo) internal() {}

var _ = unsafe.Pointer(nil)
`
	err := os.WriteFile(filepath.Join(dir, "_gen_types.go"), []byte(src), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	api, err := ExtractPackageAPI(dir)
	if err != nil {
		t.Fatal(err)
	}

	if api.Package != "testpkg" {
		t.Errorf("Package = %q, want %q", api.Package, "testpkg")
	}

	// Types.
	if len(api.Types) != 3 {
		t.Errorf("Types count = %d, want 3 (Foo, Bar, MyEnum)", len(api.Types))
	}

	foo, ok := api.Types["Foo"]
	if !ok {
		t.Fatal("missing type Foo")
	}
	if foo.Underlying != "struct" {
		t.Errorf("Foo.Underlying = %q, want %q", foo.Underlying, "struct")
	}
	if len(foo.Fields) != 1 {
		t.Errorf("Foo.Fields count = %d, want 1 (Bar)", len(foo.Fields))
	} else if foo.Fields[0].Name != "Bar" || foo.Fields[0].Type != "*Bar" {
		t.Errorf("Foo.Fields[0] = %v, want {Bar *Bar}", foo.Fields[0])
	}

	bar, ok := api.Types["Bar"]
	if !ok {
		t.Fatal("missing type Bar")
	}
	if len(bar.Fields) != 1 {
		t.Errorf("Bar.Fields count = %d, want 1 (embedded *Baz)", len(bar.Fields))
	} else if bar.Fields[0].Name != "Baz" {
		t.Errorf("Bar.Fields[0].Name = %q, want %q", bar.Fields[0].Name, "Baz")
	}

	myEnum, ok := api.Types["MyEnum"]
	if !ok {
		t.Fatal("missing type MyEnum")
	}
	if myEnum.Underlying != "int" {
		t.Errorf("MyEnum.Underlying = %q, want %q", myEnum.Underlying, "int")
	}

	// Functions.
	if len(api.Functions) != 1 {
		t.Errorf("Functions count = %d, want 1 (NewFoo)", len(api.Functions))
	}
	newFoo, ok := api.Functions["NewFoo"]
	if !ok {
		t.Fatal("missing function NewFoo")
	}
	if len(newFoo.Params) != 1 || newFoo.Params[0].Type != "string" {
		t.Errorf("NewFoo params = %v, want [(name string)]", newFoo.Params)
	}
	if len(newFoo.Returns) != 1 || newFoo.Returns[0] != "*Foo" {
		t.Errorf("NewFoo returns = %v, want [*Foo]", newFoo.Returns)
	}

	// Methods.
	if len(api.Methods) != 3 {
		t.Errorf("Methods count = %d, want 3 (GetBar, SetName, Instance)", len(api.Methods))
		for k := range api.Methods {
			t.Logf("  method: %s", k)
		}
	}

	getBar, ok := api.Methods["Foo.GetBar"]
	if !ok {
		t.Fatal("missing method Foo.GetBar")
	}
	if getBar.Receiver != "*Foo" {
		t.Errorf("GetBar.Receiver = %q, want %q", getBar.Receiver, "*Foo")
	}
	if len(getBar.Returns) != 1 || getBar.Returns[0] != "*Bar" {
		t.Errorf("GetBar returns = %v, want [*Bar]", getBar.Returns)
	}

	setName, ok := api.Methods["Foo.SetName"]
	if !ok {
		t.Fatal("missing method Foo.SetName")
	}
	if len(setName.Params) != 2 {
		t.Errorf("SetName params count = %d, want 2", len(setName.Params))
	}

	// Constants.
	if len(api.Constants) != 3 {
		t.Errorf("Constants count = %d, want 3 (EnumA, EnumB, EnumC)", len(api.Constants))
	}
	enumA, ok := api.Constants["EnumA"]
	if !ok {
		t.Fatal("missing constant EnumA")
	}
	if enumA.Type != "MyEnum" {
		t.Errorf("EnumA.Type = %q, want %q", enumA.Type, "MyEnum")
	}
	if enumA.Value != "1" {
		t.Errorf("EnumA.Value = %q, want %q", enumA.Value, "1")
	}

	// Unexported items should not appear.
	if _, ok := api.Functions["helper"]; ok {
		t.Error("unexported function 'helper' should not appear")
	}
	if _, ok := api.Methods["Foo.internal"]; ok {
		t.Error("unexported method 'internal' should not appear")
	}
}

func TestExtractRealGenFiles(t *testing.T) {
	// Test against the actual generated files in gst/.
	gstDir := findGstDir(t)
	if gstDir == "" {
		t.Skip("gst/ directory not found")
	}

	api, err := ExtractPackageAPI(gstDir)
	if err != nil {
		t.Fatal(err)
	}

	if api.Package != "gst" {
		t.Errorf("Package = %q, want %q", api.Package, "gst")
	}

	// Should have extracted a non-trivial number of symbols.
	if len(api.Types) < 3 {
		t.Errorf("Types count = %d, expected at least 3", len(api.Types))
	}
	if len(api.Constants) < 50 {
		t.Errorf("Constants count = %d, expected at least 50", len(api.Constants))
	}
	if len(api.Functions) < 5 {
		t.Errorf("Functions count = %d, expected at least 5", len(api.Functions))
	}

	t.Logf("Extracted from gst/: %d types, %d functions, %d methods, %d constants",
		len(api.Types), len(api.Functions), len(api.Methods), len(api.Constants))
}

func TestExtractSelfDiff(t *testing.T) {
	// Extract the API twice from the same directory — diff should be empty.
	gstDir := findGstDir(t)
	if gstDir == "" {
		t.Skip("gst/ directory not found")
	}

	api1, err := ExtractPackageAPI(gstDir)
	if err != nil {
		t.Fatal(err)
	}

	api2, err := ExtractPackageAPI(gstDir)
	if err != nil {
		t.Fatal(err)
	}

	report := Diff(api1, api2)
	if len(report.Changes) != 0 {
		t.Errorf("Self-diff produced %d changes, expected 0", len(report.Changes))
		for _, c := range report.Changes {
			t.Logf("  %v: %s (breaking=%v)", c.Kind, c.Symbol, c.IsBreaking)
		}
	}
}

// findGstDir walks up from the test directory to find gst/.
func findGstDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "gst")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
