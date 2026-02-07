package apidiff

import (
	"encoding/json"
	"fmt"
	"strings"
)

// HasBreakingChanges returns true if any report contains breaking changes.
func HasBreakingChanges(reports []*DiffReport) bool {
	for _, r := range reports {
		if r.HasBreaking() {
			return true
		}
	}
	return false
}

// TotalBreaking returns the total number of breaking changes across all reports.
func TotalBreaking(reports []*DiffReport) int {
	n := 0
	for _, r := range reports {
		n += r.BreakingCount()
	}
	return n
}

// TotalNonBreaking returns the total number of non-breaking changes across all reports.
func TotalNonBreaking(reports []*DiffReport) int {
	n := 0
	for _, r := range reports {
		n += r.NonBreakingCount()
	}
	return n
}

// FormatText produces a human-readable text report.
func FormatText(reports []*DiffReport) string {
	var buf strings.Builder

	for _, report := range reports {
		if len(report.Changes) == 0 {
			continue
		}
		fmt.Fprintf(&buf, "Package %s: %d breaking, %d non-breaking\n",
			report.Package, report.BreakingCount(), report.NonBreakingCount())

		for _, c := range report.Changes {
			label := "  "
			if c.IsBreaking {
				label = "! "
			}
			buf.WriteString(label + changeKindLabel(c.Kind) + " " + c.Symbol)
			if c.Old != "" && c.New != "" {
				buf.WriteString(": " + c.Old + " -> " + c.New)
			} else if c.Old != "" {
				buf.WriteString(": " + c.Old)
			} else if c.New != "" {
				buf.WriteString(": " + c.New)
			}
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	total := TotalBreaking(reports) + TotalNonBreaking(reports)
	if total == 0 {
		buf.WriteString("No API changes detected.\n")
	} else {
		fmt.Fprintf(&buf, "Total: %d breaking, %d non-breaking changes\n",
			TotalBreaking(reports), TotalNonBreaking(reports))
	}

	return buf.String()
}

// FormatMarkdown produces a markdown report suitable for a PR body.
func FormatMarkdown(reports []*DiffReport) string {
	var buf strings.Builder

	totalBreaking := TotalBreaking(reports)
	totalNonBreaking := TotalNonBreaking(reports)

	if totalBreaking == 0 && totalNonBreaking == 0 {
		buf.WriteString("No API changes detected.\n")
		return buf.String()
	}

	for _, report := range reports {
		if len(report.Changes) == 0 {
			continue
		}

		fmt.Fprintf(&buf, "### Package `%s`\n\n", report.Package)

		// Collect breaking and non-breaking changes.
		var breaking, nonBreaking []Change
		for _, c := range report.Changes {
			if c.IsBreaking {
				breaking = append(breaking, c)
			} else {
				nonBreaking = append(nonBreaking, c)
			}
		}

		if len(breaking) > 0 {
			fmt.Fprintf(&buf, "**Breaking Changes (%d)**\n\n", len(breaking))
			for _, c := range breaking {
				buf.WriteString("- **" + changeKindLabel(c.Kind) + "** `" + c.Symbol + "`")
				if c.Old != "" && c.New != "" {
					buf.WriteString(": `" + c.Old + "` -> `" + c.New + "`")
				} else if c.Old != "" {
					buf.WriteString(": `" + c.Old + "`")
				}
				buf.WriteString("\n")
			}
			buf.WriteString("\n")
		}

		if len(nonBreaking) > 0 {
			fmt.Fprintf(&buf, "**Non-Breaking Changes (%d)**\n\n", len(nonBreaking))
			for _, c := range nonBreaking {
				buf.WriteString("- **" + changeKindLabel(c.Kind) + "** `" + c.Symbol + "`")
				if c.New != "" {
					buf.WriteString(": `" + c.New + "`")
				}
				buf.WriteString("\n")
			}
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// jsonReport is the JSON serialization format.
type jsonReport struct {
	OldVersion      string       `json:"old_version"`
	NewVersion      string       `json:"new_version"`
	UpdateKind      string       `json:"update_kind"`
	Packages        []jsonPkg    `json:"packages"`
	TotalBreaking   int          `json:"total_breaking"`
	TotalNonBreaking int         `json:"total_nonbreaking"`
}

type jsonPkg struct {
	Package         string       `json:"package"`
	BreakingCount   int          `json:"breaking_count"`
	NonBreakingCount int         `json:"nonbreaking_count"`
	Changes         []jsonChange `json:"changes"`
}

type jsonChange struct {
	Kind       string `json:"kind"`
	Symbol     string `json:"symbol"`
	IsBreaking bool   `json:"is_breaking"`
	Old        string `json:"old,omitempty"`
	New        string `json:"new,omitempty"`
}

// FormatJSON produces a JSON report for CI consumption.
func FormatJSON(reports []*DiffReport, oldVer, newVer string) ([]byte, error) {
	oldV, _ := ParseVersion(oldVer)
	newV, _ := ParseVersion(newVer)
	kind := ClassifyUpdate(oldV, newV)

	jr := jsonReport{
		OldVersion:       oldVer,
		NewVersion:       newVer,
		UpdateKind:       kind.String(),
		TotalBreaking:    TotalBreaking(reports),
		TotalNonBreaking: TotalNonBreaking(reports),
	}

	for _, report := range reports {
		pkg := jsonPkg{
			Package:         report.Package,
			BreakingCount:   report.BreakingCount(),
			NonBreakingCount: report.NonBreakingCount(),
		}
		for _, c := range report.Changes {
			pkg.Changes = append(pkg.Changes, jsonChange{
				Kind:       changeKindLabel(c.Kind),
				Symbol:     c.Symbol,
				IsBreaking: c.IsBreaking,
				Old:        c.Old,
				New:        c.New,
			})
		}
		jr.Packages = append(jr.Packages, pkg)
	}

	return json.MarshalIndent(jr, "", "  ")
}

func changeKindLabel(k ChangeKind) string {
	switch k {
	case TypeAdded:
		return "Added type"
	case TypeRemoved:
		return "Removed type"
	case TypeChanged:
		return "Changed type"
	case FuncAdded:
		return "Added function"
	case FuncRemoved:
		return "Removed function"
	case FuncSignatureChanged:
		return "Changed function signature"
	case MethodAdded:
		return "Added method"
	case MethodRemoved:
		return "Removed method"
	case MethodSignatureChanged:
		return "Changed method signature"
	case ConstAdded:
		return "Added constant"
	case ConstRemoved:
		return "Removed constant"
	case ConstTypeChanged:
		return "Changed constant type"
	default:
		return "Unknown change"
	}
}
