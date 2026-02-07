package apidiff

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.24", Version{1, 24, 0}, false},
		{"1.24.3", Version{1, 24, 3}, false},
		{"2.0.0", Version{2, 0, 0}, false},
		{"1.26", Version{1, 26, 0}, false},
		{"", Version{}, true},
		{"1", Version{}, true},
		{"1.2.3.4", Version{}, true},
		{"abc.def", Version{}, true},
	}

	for _, tt := range tests {
		got, err := ParseVersion(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseVersion(%q) expected error, got %v", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseVersion(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestClassifyUpdate(t *testing.T) {
	tests := []struct {
		old, new Version
		want     UpdateKind
	}{
		{Version{1, 24, 0}, Version{1, 24, 1}, PatchUpdate},
		{Version{1, 24, 1}, Version{1, 24, 3}, PatchUpdate},
		{Version{1, 24, 0}, Version{1, 26, 0}, MinorUpdate},
		{Version{1, 24, 3}, Version{1, 26, 0}, MinorUpdate},
		{Version{1, 24, 0}, Version{2, 0, 0}, MajorUpdate},
	}

	for _, tt := range tests {
		got := ClassifyUpdate(tt.old, tt.new)
		if got != tt.want {
			t.Errorf("ClassifyUpdate(%v, %v) = %v, want %v", tt.old, tt.new, got, tt.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 24, 0}, "1.24"},
		{Version{1, 24, 3}, "1.24.3"},
		{Version{2, 0, 0}, "2.0"},
	}

	for _, tt := range tests {
		got := tt.v.String()
		if got != tt.want {
			t.Errorf("Version%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestUpdateKindString(t *testing.T) {
	tests := []struct {
		k    UpdateKind
		want string
	}{
		{PatchUpdate, "patch"},
		{MinorUpdate, "minor"},
		{MajorUpdate, "major"},
	}

	for _, tt := range tests {
		got := tt.k.String()
		if got != tt.want {
			t.Errorf("UpdateKind(%d).String() = %q, want %q", tt.k, got, tt.want)
		}
	}
}
