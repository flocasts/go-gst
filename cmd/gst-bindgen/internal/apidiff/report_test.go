package apidiff

import (
	"encoding/json"
	"strings"
	"testing"
)

func sampleReports() []*DiffReport {
	return []*DiffReport{
		{
			Package: "gst",
			Changes: []Change{
				{Kind: TypeRemoved, Symbol: "OldType", Old: "type OldType struct{}", IsBreaking: true},
				{Kind: FuncAdded, Symbol: "NewFunc", New: "func NewFunc() *Widget", IsBreaking: false},
				{Kind: MethodSignatureChanged, Symbol: "Foo.GetName",
					Old: "func (*Foo) GetName() string", New: "func (*Foo) GetName() (string, error)", IsBreaking: true},
			},
		},
		{
			Package: "video",
			Changes: []Change{
				{Kind: ConstAdded, Symbol: "NewConst", New: "VideoFormat = C.GST_VIDEO_FORMAT_NEW", IsBreaking: false},
			},
		},
	}
}

func TestFormatText(t *testing.T) {
	text := FormatText(sampleReports())

	if !strings.Contains(text, "Package gst: 2 breaking, 1 non-breaking") {
		t.Errorf("missing gst summary in:\n%s", text)
	}
	if !strings.Contains(text, "! Removed type OldType") {
		t.Errorf("missing breaking type removal in:\n%s", text)
	}
	if !strings.Contains(text, "Total: 2 breaking, 2 non-breaking") {
		t.Errorf("missing total summary in:\n%s", text)
	}
}

func TestFormatTextEmpty(t *testing.T) {
	text := FormatText([]*DiffReport{{Package: "gst"}})
	if !strings.Contains(text, "No API changes") {
		t.Errorf("expected 'No API changes' in:\n%s", text)
	}
}

func TestFormatMarkdown(t *testing.T) {
	md := FormatMarkdown(sampleReports())

	if !strings.Contains(md, "### Package `gst`") {
		t.Errorf("missing package header in:\n%s", md)
	}
	if !strings.Contains(md, "**Breaking Changes (2)**") {
		t.Errorf("missing breaking section in:\n%s", md)
	}
	if !strings.Contains(md, "**Removed type** `OldType`") {
		t.Errorf("missing type removal in:\n%s", md)
	}
	if !strings.Contains(md, "**Added function** `NewFunc`") {
		t.Errorf("missing function addition in:\n%s", md)
	}
}

func TestFormatMarkdownEmpty(t *testing.T) {
	md := FormatMarkdown([]*DiffReport{{Package: "gst"}})
	if !strings.Contains(md, "No API changes") {
		t.Errorf("expected 'No API changes' in:\n%s", md)
	}
}

func TestFormatJSON(t *testing.T) {
	data, err := FormatJSON(sampleReports(), "1.24", "1.26")
	if err != nil {
		t.Fatal(err)
	}

	var report jsonReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}

	if report.OldVersion != "1.24" {
		t.Errorf("OldVersion = %q, want %q", report.OldVersion, "1.24")
	}
	if report.NewVersion != "1.26" {
		t.Errorf("NewVersion = %q, want %q", report.NewVersion, "1.26")
	}
	if report.UpdateKind != "minor" {
		t.Errorf("UpdateKind = %q, want %q", report.UpdateKind, "minor")
	}
	if report.TotalBreaking != 2 {
		t.Errorf("TotalBreaking = %d, want 2", report.TotalBreaking)
	}
	if report.TotalNonBreaking != 2 {
		t.Errorf("TotalNonBreaking = %d, want 2", report.TotalNonBreaking)
	}
	if len(report.Packages) != 2 {
		t.Errorf("Packages count = %d, want 2", len(report.Packages))
	}
}

func TestHasBreakingChanges(t *testing.T) {
	if !HasBreakingChanges(sampleReports()) {
		t.Error("expected breaking changes")
	}

	noBreaking := []*DiffReport{{
		Package: "test",
		Changes: []Change{{Kind: FuncAdded, Symbol: "F", IsBreaking: false}},
	}}
	if HasBreakingChanges(noBreaking) {
		t.Error("should not have breaking changes")
	}
}
