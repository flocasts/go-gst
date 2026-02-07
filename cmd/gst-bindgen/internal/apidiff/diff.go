package apidiff

import (
	"sort"
	"strings"
)

// Diff compares two PackageAPI snapshots and returns a report of changes.
func Diff(old, new *PackageAPI) *DiffReport {
	report := &DiffReport{Package: old.Package}

	diffTypes(report, old.Types, new.Types)
	diffFunctions(report, old.Functions, new.Functions)
	diffMethods(report, old.Methods, new.Methods)
	diffConstants(report, old.Constants, new.Constants)

	// Sort changes for deterministic output.
	sort.Slice(report.Changes, func(i, j int) bool {
		if report.Changes[i].Kind != report.Changes[j].Kind {
			return report.Changes[i].Kind < report.Changes[j].Kind
		}
		return report.Changes[i].Symbol < report.Changes[j].Symbol
	})

	return report
}

func diffTypes(report *DiffReport, old, new map[string]TypeDef) {
	for name, oldT := range old {
		newT, ok := new[name]
		if !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       TypeRemoved,
				Symbol:     name,
				Old:        formatTypeDef(oldT),
				IsBreaking: true,
			})
			continue
		}
		if oldT.Underlying != newT.Underlying {
			report.Changes = append(report.Changes, Change{
				Kind:       TypeChanged,
				Symbol:     name,
				Old:        formatTypeDef(oldT),
				New:        formatTypeDef(newT),
				IsBreaking: true,
			})
			continue
		}
		// Check for removed exported fields (breaking).
		if oldT.Underlying == "struct" {
			removedFields := fieldDiff(oldT.Fields, newT.Fields)
			for _, f := range removedFields {
				report.Changes = append(report.Changes, Change{
					Kind:       TypeChanged,
					Symbol:     name + "." + f.Name,
					Old:        f.Name + " " + f.Type,
					IsBreaking: true,
				})
			}
		}
	}

	for name, newT := range new {
		if _, ok := old[name]; !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       TypeAdded,
				Symbol:     name,
				New:        formatTypeDef(newT),
				IsBreaking: false,
			})
		}
	}
}

func diffFunctions(report *DiffReport, old, new map[string]FuncSig) {
	for name, oldF := range old {
		newF, ok := new[name]
		if !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       FuncRemoved,
				Symbol:     name,
				Old:        formatFuncSig(oldF),
				IsBreaking: true,
			})
			continue
		}
		if !signaturesEqual(oldF, newF) {
			report.Changes = append(report.Changes, Change{
				Kind:       FuncSignatureChanged,
				Symbol:     name,
				Old:        formatFuncSig(oldF),
				New:        formatFuncSig(newF),
				IsBreaking: true,
			})
		}
	}

	for name, newF := range new {
		if _, ok := old[name]; !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       FuncAdded,
				Symbol:     name,
				New:        formatFuncSig(newF),
				IsBreaking: false,
			})
		}
	}
}

func diffMethods(report *DiffReport, old, new map[string]FuncSig) {
	for key, oldM := range old {
		newM, ok := new[key]
		if !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       MethodRemoved,
				Symbol:     key,
				Old:        formatMethodSig(oldM),
				IsBreaking: true,
			})
			continue
		}
		if !signaturesEqual(oldM, newM) {
			report.Changes = append(report.Changes, Change{
				Kind:       MethodSignatureChanged,
				Symbol:     key,
				Old:        formatMethodSig(oldM),
				New:        formatMethodSig(newM),
				IsBreaking: true,
			})
		}
	}

	for key, newM := range new {
		if _, ok := old[key]; !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       MethodAdded,
				Symbol:     key,
				New:        formatMethodSig(newM),
				IsBreaking: false,
			})
		}
	}
}

func diffConstants(report *DiffReport, old, new map[string]ConstDef) {
	for name, oldC := range old {
		newC, ok := new[name]
		if !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       ConstRemoved,
				Symbol:     name,
				Old:        oldC.Type + " = " + oldC.Value,
				IsBreaking: true,
			})
			continue
		}
		if oldC.Type != newC.Type {
			report.Changes = append(report.Changes, Change{
				Kind:       ConstTypeChanged,
				Symbol:     name,
				Old:        oldC.Type,
				New:        newC.Type,
				IsBreaking: true,
			})
		}
	}

	for name, newC := range new {
		if _, ok := old[name]; !ok {
			report.Changes = append(report.Changes, Change{
				Kind:       ConstAdded,
				Symbol:     name,
				New:        newC.Type + " = " + newC.Value,
				IsBreaking: false,
			})
		}
	}
}

// signaturesEqual compares two function signatures by parameter types and return types.
// Parameter names are not considered.
func signaturesEqual(a, b FuncSig) bool {
	if len(a.Params) != len(b.Params) || len(a.Returns) != len(b.Returns) {
		return false
	}
	for i := range a.Params {
		if a.Params[i].Type != b.Params[i].Type {
			return false
		}
	}
	for i := range a.Returns {
		if a.Returns[i] != b.Returns[i] {
			return false
		}
	}
	return true
}

// fieldDiff returns fields in old that are missing from new.
func fieldDiff(old, new []Field) []Field {
	newSet := make(map[string]string)
	for _, f := range new {
		newSet[f.Name] = f.Type
	}
	var removed []Field
	for _, f := range old {
		newType, ok := newSet[f.Name]
		if !ok || newType != f.Type {
			removed = append(removed, f)
		}
	}
	return removed
}

func formatTypeDef(td TypeDef) string {
	if td.Underlying == "struct" {
		var fields []string
		for _, f := range td.Fields {
			fields = append(fields, f.Name+" "+f.Type)
		}
		if len(fields) == 0 {
			return "type " + td.Name + " struct{}"
		}
		return "type " + td.Name + " struct{ " + strings.Join(fields, "; ") + " }"
	}
	return "type " + td.Name + " " + td.Underlying
}

func formatFuncSig(fs FuncSig) string {
	var buf strings.Builder
	buf.WriteString("func " + fs.Name + "(")
	for i, p := range fs.Params {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(p.Type)
	}
	buf.WriteString(")")
	if len(fs.Returns) > 0 {
		if len(fs.Returns) == 1 {
			buf.WriteString(" " + fs.Returns[0])
		} else {
			buf.WriteString(" (" + strings.Join(fs.Returns, ", ") + ")")
		}
	}
	return buf.String()
}

func formatMethodSig(fs FuncSig) string {
	var buf strings.Builder
	buf.WriteString("func (" + fs.Receiver + ") " + fs.Name + "(")
	for i, p := range fs.Params {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(p.Type)
	}
	buf.WriteString(")")
	if len(fs.Returns) > 0 {
		if len(fs.Returns) == 1 {
			buf.WriteString(" " + fs.Returns[0])
		} else {
			buf.WriteString(" (" + strings.Join(fs.Returns, ", ") + ")")
		}
	}
	return buf.String()
}
