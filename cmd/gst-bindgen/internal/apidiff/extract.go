package apidiff

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// ExtractPackageAPI parses all _gen_*.go files in dir and returns
// the exported API surface.
func ExtractPackageAPI(dir string) (*PackageAPI, error) {
	fset := token.NewFileSet()

	// Collect _gen_*.go files in dir.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	api := NewPackageAPI(filepath.Base(dir))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "_gen_") || !strings.HasSuffix(name, ".go") {
			continue
		}

		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			// Skip files with parse errors (e.g., generated code with bugs).
			fmt.Fprintf(os.Stderr, "  warning: skipping %s: %v\n", name, err)
			continue
		}

		if api.Package == filepath.Base(dir) && f.Name != nil {
			api.Package = f.Name.Name
		}

		extractFile(api, f)
	}

	return api, nil
}

func extractFile(api *PackageAPI, f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			extractGenDecl(api, d)
		case *ast.FuncDecl:
			extractFuncDecl(api, d)
		}
	}
}

func extractGenDecl(api *PackageAPI, decl *ast.GenDecl) {
	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !isExported(ts.Name.Name) {
				continue
			}
			api.Types[ts.Name.Name] = extractTypeDef(ts)
		}
	case token.CONST:
		// Track the current iota type for untyped constants in a const block.
		var blockType string
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			// If this const spec has an explicit type, use it.
			if vs.Type != nil {
				blockType = typeExprString(vs.Type)
			}
			for i, name := range vs.Names {
				if !isExported(name.Name) {
					continue
				}
				cd := ConstDef{
					Name: name.Name,
					Type: blockType,
				}
				if i < len(vs.Values) {
					cd.Value = exprString(vs.Values[i])
				}
				api.Constants[name.Name] = cd
			}
		}
	}
}

func extractTypeDef(ts *ast.TypeSpec) TypeDef {
	td := TypeDef{Name: ts.Name.Name}

	switch t := ts.Type.(type) {
	case *ast.StructType:
		td.Underlying = "struct"
		if t.Fields != nil {
			for _, field := range t.Fields.List {
				fieldType := typeExprString(field.Type)
				if len(field.Names) == 0 {
					// Embedded field — use the type name as the field name.
					embName := embeddedFieldName(field.Type)
					if isExported(embName) {
						td.Fields = append(td.Fields, Field{Name: embName, Type: fieldType})
					}
				} else {
					for _, name := range field.Names {
						if isExported(name.Name) {
							td.Fields = append(td.Fields, Field{Name: name.Name, Type: fieldType})
						}
					}
				}
			}
		}
	case *ast.FuncType:
		td.Underlying = "func" + funcTypeString(t)
	case *ast.Ident:
		td.Underlying = t.Name
	case *ast.SelectorExpr:
		td.Underlying = typeExprString(t)
	default:
		td.Underlying = typeExprString(ts.Type)
	}

	return td
}

func extractFuncDecl(api *PackageAPI, fn *ast.FuncDecl) {
	if !isExported(fn.Name.Name) {
		return
	}

	sig := FuncSig{Name: fn.Name.Name}

	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recvType := typeExprString(fn.Recv.List[0].Type)
		sig.Receiver = recvType
	}

	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			paramType := typeExprString(field.Type)
			if len(field.Names) == 0 {
				sig.Params = append(sig.Params, Param{Type: paramType})
			} else {
				for _, name := range field.Names {
					sig.Params = append(sig.Params, Param{Name: name.Name, Type: paramType})
				}
			}
		}
	}

	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			retType := typeExprString(field.Type)
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for range count {
				sig.Returns = append(sig.Returns, retType)
			}
		}
	}

	if sig.Receiver != "" {
		// Strip pointer from receiver for the key.
		recvName := strings.TrimPrefix(sig.Receiver, "*")
		key := recvName + "." + fn.Name.Name
		api.Methods[key] = sig
	} else {
		api.Functions[fn.Name.Name] = sig
	}
}

// typeExprString converts an AST type expression to a string representation.
func typeExprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return typeExprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + typeExprString(e.X)
	case *ast.ArrayType:
		if e.Len == nil {
			return "[]" + typeExprString(e.Elt)
		}
		return "[" + exprString(e.Len) + "]" + typeExprString(e.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(e.Key) + "]" + typeExprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func" + funcTypeString(e)
	case *ast.Ellipsis:
		return "..." + typeExprString(e.Elt)
	case *ast.ChanType:
		switch e.Dir {
		case ast.SEND:
			return "chan<- " + typeExprString(e.Value)
		case ast.RECV:
			return "<-chan " + typeExprString(e.Value)
		default:
			return "chan " + typeExprString(e.Value)
		}
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// exprString converts an arbitrary AST expression to a string (for const values).
func exprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.BinaryExpr:
		return exprString(e.X) + " " + e.Op.String() + " " + exprString(e.Y)
	case *ast.UnaryExpr:
		return e.Op.String() + exprString(e.X)
	case *ast.ParenExpr:
		return "(" + exprString(e.X) + ")"
	case *ast.CallExpr:
		var args []string
		for _, arg := range e.Args {
			args = append(args, exprString(arg))
		}
		return exprString(e.Fun) + "(" + strings.Join(args, ", ") + ")"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func funcTypeString(ft *ast.FuncType) string {
	var buf strings.Builder
	buf.WriteString("(")
	if ft.Params != nil {
		first := true
		for _, field := range ft.Params.List {
			paramType := typeExprString(field.Type)
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for range count {
				if !first {
					buf.WriteString(", ")
				}
				buf.WriteString(paramType)
				first = false
			}
		}
	}
	buf.WriteString(")")

	if ft.Results != nil && len(ft.Results.List) > 0 {
		var rets []string
		for _, field := range ft.Results.List {
			retType := typeExprString(field.Type)
			count := len(field.Names)
			if count == 0 {
				count = 1
			}
			for range count {
				rets = append(rets, retType)
			}
		}
		if len(rets) == 1 {
			buf.WriteString(" " + rets[0])
		} else {
			buf.WriteString(" (" + strings.Join(rets, ", ") + ")")
		}
	}

	return buf.String()
}

// embeddedFieldName extracts the type name from an embedded field expression.
func embeddedFieldName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return embeddedFieldName(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name
	default:
		return ""
	}
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}
