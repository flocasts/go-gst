package apidiff

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version with major, minor, and patch components.
type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	if v.Patch == 0 {
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// UpdateKind classifies the type of version update.
type UpdateKind int

const (
	PatchUpdate UpdateKind = iota
	MinorUpdate
	MajorUpdate
)

func (k UpdateKind) String() string {
	switch k {
	case PatchUpdate:
		return "patch"
	case MinorUpdate:
		return "minor"
	case MajorUpdate:
		return "major"
	}
	return "unknown"
}

// ParseVersion parses a version string like "1.24" or "1.24.3".
func ParseVersion(s string) (Version, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q: expected major.minor or major.minor.patch", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version in %q: %w", s, err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version in %q: %w", s, err)
	}

	patch := 0
	if len(parts) == 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version in %q: %w", s, err)
		}
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// ClassifyUpdate determines whether the update from old to new is a major,
// minor, or patch release.
func ClassifyUpdate(old, new Version) UpdateKind {
	if old.Major != new.Major {
		return MajorUpdate
	}
	if old.Minor != new.Minor {
		return MinorUpdate
	}
	return PatchUpdate
}
