package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

const (
	fileName       = "reset.gen.go"
	generateMarker = "// generate:reset"
)

const (
	fieldKindInt       = "int"
	fieldKindString    = "string"
	fieldKindPtrString = "ptrString"
	fieldKindSlice     = "slice"
	fieldKindMap       = "map"
	fieldKindPtrStruct = "ptrStruct"
)

//go:embed reset.tmpl
var resetTemplate string

type fieldInfo struct {
	Name string
	Kind string
}

type structInfo struct {
	Name   string
	Fields []fieldInfo
}

type fileTemplateData struct {
	Package string
	Structs []structInfo
}

var (
	resetTmpl = template.Must(template.New("reset").Parse(resetTemplate))
)

func commentGroupHasComment(cg *ast.CommentGroup) bool {
	for _, c := range cg.List {
		if strings.Contains(c.Text, generateMarker) {
			return true
		}
	}
	return false
}

func hasComment(ts *ast.TypeSpec, genDecl *ast.GenDecl) bool {
	if ts.Doc != nil && commentGroupHasComment(ts.Doc) {
		return true
	}

	if genDecl.Doc != nil && commentGroupHasComment(genDecl.Doc) {
		return true
	}

	return false
}

func getStructs(pkg *packages.Package) []structInfo {
	var result []structInfo
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				var fields []fieldInfo
				for _, f := range st.Fields.List {
					if len(f.Names) == 0 {
						continue
					}
					kind := detectFieldKind(f.Type, ts.Name.Name)
					if kind == "" {
						continue
					}

					for _, nameIdent := range f.Names {
						fields = append(fields, fieldInfo{
							Name: nameIdent.Name,
							Kind: kind,
						})
					}
				}

				if !hasComment(ts, genDecl) {
					continue
				}

				result = append(result, structInfo{
					Name:   ts.Name.Name,
					Fields: fields,
				})
			}
		}
	}
	return result
}

func detectFieldKind(expr ast.Expr, structName string) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case fieldKindInt:
			return fieldKindInt
		case fieldKindString:
			return fieldKindString
		}
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			switch id.Name {
			case fieldKindString:
				return fieldKindPtrString
			case structName:
				return fieldKindPtrStruct
			}
		}
	case *ast.ArrayType:
		if t.Len == nil {
			return fieldKindSlice
		}
	case *ast.MapType:
		return fieldKindMap
	}

	return ""
}

func generateFile(pkgName string, structs []structInfo) ([]byte, error) {
	data := fileTemplateData{
		Package: pkgName,
		Structs: structs,
	}

	var buf bytes.Buffer
	if err := resetTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

func processPackage(pkg *packages.Package) error {

	if len(pkg.GoFiles) == 0 {
		return nil
	}

	structs := getStructs(pkg)
	if len(structs) == 0 {
		return nil
	}

	dir := filepath.Dir(pkg.GoFiles[0])
	src, err := generateFile(pkg.Name, structs)
	if err != nil {
		return fmt.Errorf("generate file: %w", err)
	}

	outPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	return nil
}

func run() error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	config := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax,
		Dir: wd,
	}

	patterns := []string{
		"./...",
		"./cmd/reset/testdata",
	}

	pkgs, err := packages.Load(config, patterns...)
	if err != nil {
		return fmt.Errorf("packages.Load: %w", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	for _, pkg := range pkgs {
		if err := processPackage(pkg); err != nil {
			return fmt.Errorf("process package %q: %w", pkg.PkgPath, err)
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "reset generator error:", err)
		os.Exit(1)
	}
}
