package apidiff

import "testing"

func TestDiffIdentical(t *testing.T) {
	api := &PackageAPI{
		Package: "test",
		Types: map[string]TypeDef{
			"Foo": {Name: "Foo", Underlying: "struct", Fields: []Field{{Name: "X", Type: "int"}}},
		},
		Functions: map[string]FuncSig{
			"NewFoo": {Name: "NewFoo", Returns: []string{"*Foo"}},
		},
		Methods: map[string]FuncSig{
			"Foo.GetX": {Name: "GetX", Receiver: "*Foo", Returns: []string{"int"}},
		},
		Constants: map[string]ConstDef{
			"ConstA": {Name: "ConstA", Type: "int", Value: "1"},
		},
	}

	report := Diff(api, api)
	if len(report.Changes) != 0 {
		t.Errorf("identical APIs produced %d changes", len(report.Changes))
	}
}

func TestDiffTypeRemoved(t *testing.T) {
	old := NewPackageAPI("test")
	old.Types["Foo"] = TypeDef{Name: "Foo", Underlying: "struct"}

	new := NewPackageAPI("test")

	report := Diff(old, new)
	if len(report.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(report.Changes))
	}
	c := report.Changes[0]
	if c.Kind != TypeRemoved || c.Symbol != "Foo" || !c.IsBreaking {
		t.Errorf("unexpected change: %+v", c)
	}
}

func TestDiffTypeAdded(t *testing.T) {
	old := NewPackageAPI("test")

	new := NewPackageAPI("test")
	new.Types["Bar"] = TypeDef{Name: "Bar", Underlying: "struct"}

	report := Diff(old, new)
	if len(report.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(report.Changes))
	}
	c := report.Changes[0]
	if c.Kind != TypeAdded || c.Symbol != "Bar" || c.IsBreaking {
		t.Errorf("unexpected change: %+v", c)
	}
}

func TestDiffTypeUnderlyingChanged(t *testing.T) {
	old := NewPackageAPI("test")
	old.Types["MyEnum"] = TypeDef{Name: "MyEnum", Underlying: "int"}

	new := NewPackageAPI("test")
	new.Types["MyEnum"] = TypeDef{Name: "MyEnum", Underlying: "uint"}

	report := Diff(old, new)
	if len(report.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(report.Changes))
	}
	c := report.Changes[0]
	if c.Kind != TypeChanged || !c.IsBreaking {
		t.Errorf("unexpected change: %+v", c)
	}
}

func TestDiffFuncRemoved(t *testing.T) {
	old := NewPackageAPI("test")
	old.Functions["DoThing"] = FuncSig{Name: "DoThing"}

	new := NewPackageAPI("test")

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking, got %d", report.BreakingCount())
	}
}

func TestDiffFuncAdded(t *testing.T) {
	old := NewPackageAPI("test")

	new := NewPackageAPI("test")
	new.Functions["DoThing"] = FuncSig{Name: "DoThing"}

	report := Diff(old, new)
	if report.NonBreakingCount() != 1 {
		t.Errorf("expected 1 non-breaking, got %d", report.NonBreakingCount())
	}
	if report.HasBreaking() {
		t.Error("should not have breaking changes")
	}
}

func TestDiffFuncSignatureChanged(t *testing.T) {
	old := NewPackageAPI("test")
	old.Functions["DoThing"] = FuncSig{
		Name:    "DoThing",
		Params:  []Param{{Name: "x", Type: "int"}},
		Returns: []string{"string"},
	}

	new := NewPackageAPI("test")
	new.Functions["DoThing"] = FuncSig{
		Name:    "DoThing",
		Params:  []Param{{Name: "x", Type: "int"}},
		Returns: []string{"string", "error"},
	}

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking, got %d", report.BreakingCount())
	}
	if report.Changes[0].Kind != FuncSignatureChanged {
		t.Errorf("expected FuncSignatureChanged, got %v", report.Changes[0].Kind)
	}
}

func TestDiffMethodChanges(t *testing.T) {
	old := NewPackageAPI("test")
	old.Methods["Foo.GetX"] = FuncSig{Name: "GetX", Receiver: "*Foo", Returns: []string{"int"}}
	old.Methods["Foo.SetX"] = FuncSig{Name: "SetX", Receiver: "*Foo", Params: []Param{{Type: "int"}}}

	new := NewPackageAPI("test")
	// GetX removed
	new.Methods["Foo.SetX"] = FuncSig{Name: "SetX", Receiver: "*Foo", Params: []Param{{Type: "int"}}}
	// NewMethod added
	new.Methods["Foo.GetY"] = FuncSig{Name: "GetY", Receiver: "*Foo", Returns: []string{"int"}}

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking (GetX removed), got %d", report.BreakingCount())
	}
	if report.NonBreakingCount() != 1 {
		t.Errorf("expected 1 non-breaking (GetY added), got %d", report.NonBreakingCount())
	}
}

func TestDiffConstRemoved(t *testing.T) {
	old := NewPackageAPI("test")
	old.Constants["FooConst"] = ConstDef{Name: "FooConst", Type: "int", Value: "42"}

	new := NewPackageAPI("test")

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking, got %d", report.BreakingCount())
	}
}

func TestDiffConstTypeChanged(t *testing.T) {
	old := NewPackageAPI("test")
	old.Constants["X"] = ConstDef{Name: "X", Type: "int", Value: "1"}

	new := NewPackageAPI("test")
	new.Constants["X"] = ConstDef{Name: "X", Type: "uint", Value: "1"}

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking, got %d", report.BreakingCount())
	}
	if report.Changes[0].Kind != ConstTypeChanged {
		t.Errorf("expected ConstTypeChanged, got %v", report.Changes[0].Kind)
	}
}

func TestDiffMixed(t *testing.T) {
	old := NewPackageAPI("test")
	old.Types["A"] = TypeDef{Name: "A", Underlying: "struct"}
	old.Functions["F1"] = FuncSig{Name: "F1"}
	old.Constants["C1"] = ConstDef{Name: "C1", Type: "int", Value: "1"}

	new := NewPackageAPI("test")
	new.Types["A"] = TypeDef{Name: "A", Underlying: "struct"} // unchanged
	new.Types["B"] = TypeDef{Name: "B", Underlying: "struct"} // added
	// F1 removed
	new.Functions["F2"] = FuncSig{Name: "F2"} // added
	new.Constants["C1"] = ConstDef{Name: "C1", Type: "int", Value: "1"} // unchanged
	new.Constants["C2"] = ConstDef{Name: "C2", Type: "int", Value: "2"} // added

	report := Diff(old, new)
	if report.BreakingCount() != 1 { // F1 removed
		t.Errorf("expected 1 breaking, got %d", report.BreakingCount())
	}
	if report.NonBreakingCount() != 3 { // B added, F2 added, C2 added
		t.Errorf("expected 3 non-breaking, got %d", report.NonBreakingCount())
	}
}

func TestDiffParamNameChangeNotBreaking(t *testing.T) {
	old := NewPackageAPI("test")
	old.Functions["F"] = FuncSig{
		Name:   "F",
		Params: []Param{{Name: "oldName", Type: "string"}},
	}

	new := NewPackageAPI("test")
	new.Functions["F"] = FuncSig{
		Name:   "F",
		Params: []Param{{Name: "newName", Type: "string"}},
	}

	report := Diff(old, new)
	if len(report.Changes) != 0 {
		t.Errorf("param name change should not be a change, got %d", len(report.Changes))
	}
}

func TestDiffFieldRemoved(t *testing.T) {
	old := NewPackageAPI("test")
	old.Types["S"] = TypeDef{
		Name:       "S",
		Underlying: "struct",
		Fields:     []Field{{Name: "X", Type: "int"}, {Name: "Y", Type: "string"}},
	}

	new := NewPackageAPI("test")
	new.Types["S"] = TypeDef{
		Name:       "S",
		Underlying: "struct",
		Fields:     []Field{{Name: "X", Type: "int"}},
	}

	report := Diff(old, new)
	if report.BreakingCount() != 1 {
		t.Errorf("expected 1 breaking (field Y removed), got %d", report.BreakingCount())
	}
}
