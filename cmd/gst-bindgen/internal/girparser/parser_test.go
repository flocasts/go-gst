package girparser

import (
	"os"
	"path/filepath"
	"testing"
)

func girFilesDir() string {
	// Walk up from the test file to find gir-files/
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, "gir-files")
		if _, err := os.Stat(candidate); err == nil {
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

func TestParseGst(t *testing.T) {
	girDir := girFilesDir()
	if girDir == "" {
		t.Skip("gir-files directory not found")
	}

	repo, err := ParseRepository(filepath.Join(girDir, "Gst-1.0.gir"))
	if err != nil {
		t.Fatalf("ParseRepository: %v", err)
	}

	ns := &repo.Namespace
	if ns.Name != "Gst" {
		t.Errorf("namespace name = %q, want %q", ns.Name, "Gst")
	}
	if ns.Version != "1.0" {
		t.Errorf("namespace version = %q, want %q", ns.Version, "1.0")
	}

	// Should have many classes.
	if len(ns.Classes) < 10 {
		t.Errorf("expected at least 10 classes, got %d", len(ns.Classes))
	}

	// Should have many enumerations.
	if len(ns.Enumerations) < 10 {
		t.Errorf("expected at least 10 enumerations, got %d", len(ns.Enumerations))
	}

	// Check a known class.
	var foundElement bool
	for _, cls := range ns.Classes {
		if cls.Name == "Element" {
			foundElement = true
			if cls.Parent != "Object" {
				t.Errorf("Element parent = %q, want %q", cls.Parent, "Object")
			}
			if len(cls.Methods) == 0 {
				t.Error("Element has no methods")
			}
			break
		}
	}
	if !foundElement {
		t.Error("Element class not found")
	}

	// Check a known enum.
	var foundState bool
	for _, enum := range ns.Enumerations {
		if enum.Name == "State" {
			foundState = true
			if len(enum.Members) < 4 {
				t.Errorf("State enum has %d members, expected at least 4", len(enum.Members))
			}
			break
		}
	}
	if !foundState {
		t.Error("State enumeration not found")
	}

	t.Logf("Parsed Gst-1.0: %d classes, %d records, %d interfaces, %d enums, %d bitfields, %d functions, %d callbacks",
		len(ns.Classes), len(ns.Records), len(ns.Interfaces), len(ns.Enumerations),
		len(ns.Bitfields), len(ns.Functions), len(ns.Callbacks))
}

func TestTypeRegistry(t *testing.T) {
	girDir := girFilesDir()
	if girDir == "" {
		t.Skip("gir-files directory not found")
	}

	registry := NewTypeRegistry()

	// Parse core GIR files.
	for _, filename := range []string{"GObject-2.0.gir", "Gst-1.0.gir"} {
		path := filepath.Join(girDir, filename)
		repo, err := ParseRepository(path)
		if err != nil {
			t.Fatalf("ParseRepository(%s): %v", filename, err)
		}
		registry.AddRepository(repo)
	}

	if err := registry.Resolve(); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// Element should be classified as GObject.
	elemRT := registry.LookupType("Gst.Element")
	if elemRT == nil {
		t.Fatal("Gst.Element not found in registry")
	}
	if elemRT.Kind != TypeKindGObject {
		t.Errorf("Gst.Element kind = %v, want GObject", elemRT.Kind)
	}

	// Pipeline should be GObject with parent chain through Bin -> Element -> Object.
	pipRT := registry.LookupType("Gst.Pipeline")
	if pipRT == nil {
		t.Fatal("Gst.Pipeline not found in registry")
	}
	if pipRT.Kind != TypeKindGObject {
		t.Errorf("Gst.Pipeline kind = %v, want GObject", pipRT.Kind)
	}
	t.Logf("Pipeline parent chain: %v", pipRT.ParentChain)

	// State should be an enum.
	stateRT := registry.LookupType("Gst.State")
	if stateRT == nil {
		t.Fatal("Gst.State not found in registry")
	}
	if stateRT.Kind != TypeKindEnum {
		t.Errorf("Gst.State kind = %v, want Enum", stateRT.Kind)
	}

	// Check some counts.
	gstTypes := registry.TypesInNamespace("Gst")
	t.Logf("Total types in Gst namespace: %d", len(gstTypes))

	var gobjects, miniobjects, enums, records int
	for _, rt := range gstTypes {
		switch rt.Kind {
		case TypeKindGObject:
			gobjects++
		case TypeKindMiniObject:
			miniobjects++
		case TypeKindEnum:
			enums++
		case TypeKindRecord:
			records++
		}
	}
	t.Logf("GObjects: %d, MiniObjects: %d, Enums: %d, Records: %d", gobjects, miniobjects, enums, records)
}
