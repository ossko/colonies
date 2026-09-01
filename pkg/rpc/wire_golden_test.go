package rpc

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var updateGolden = flag.Bool("update-golden", false, "regenerate the wire format golden file")

const goldenPath = "testdata/wire_format.golden"

// TestWireFormatGolden guards the RPC wire format. It extracts every JSON
// field name from every struct in this package, plus every payload type
// constant, and compares the result against a committed golden file.
//
// The wire RPC API is a compatibility contract with external executors and
// SDKs: renaming or removing a field or payload type breaks them. If this
// test fails, either revert the wire change, or, if the change is intentional
// and additive, regenerate the golden file with:
//
//	go test ./pkg/rpc -run TestWireFormatGolden -update-golden
//
// and call out the wire change explicitly in the pull request.
func TestWireFormatGolden(t *testing.T) {
	// The wire format is defined by the rpc envelope and message structs plus
	// the core domain types they embed
	current, err := extractWireFormat(".", "../core")
	assert.Nil(t, err)

	if *updateGolden {
		err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
		assert.Nil(t, err)
		err = os.WriteFile(goldenPath, []byte(current), 0644)
		assert.Nil(t, err)
		t.Log("golden file regenerated")
		return
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read %s (run with -update-golden to create it): %v", goldenPath, err)
	}
	golden := string(goldenBytes)

	if current != golden {
		t.Errorf("RPC wire format changed.\n\n%s\n\nIf this change is intentional, regenerate with:\n  go test ./pkg/rpc -run TestWireFormatGolden -update-golden", diffLines(golden, current))
	}
}

// extractWireFormat parses all non-test Go files in the given directories and
// renders a stable description of the wire format: struct fields with their
// json tags, and all payload type constants.
func extractWireFormat(dirs ...string) (string, error) {
	structs := make(map[string][]string)
	consts := make(map[string]string)

	for _, dir := range dirs {
		err := extractFromDir(dir, structs, consts)
		if err != nil {
			return "", err
		}
	}

	return renderWireFormat(structs, consts), nil
}

func extractFromDir(dir string, structs map[string][]string, consts map[string]string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return err
	}

	pkgTag := filepath.Base(dir)
	if pkgTag == "." {
		pkgTag = "rpc"
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				switch decl := n.(type) {
				case *ast.TypeSpec:
					structType, ok := decl.Type.(*ast.StructType)
					if !ok {
						return true
					}
					var fields []string
					for _, field := range structType.Fields.List {
						jsonName := ""
						if field.Tag != nil {
							tag := strings.Trim(field.Tag.Value, "`")
							jsonName = reflect.StructTag(tag).Get("json")
							jsonName = strings.Split(jsonName, ",")[0]
						}
						for _, name := range field.Names {
							fields = append(fields, fmt.Sprintf("%s json:%q", name.Name, jsonName))
						}
						if len(field.Names) == 0 {
							// Embedded field
							fields = append(fields, fmt.Sprintf("embedded:%s json:%q", exprString(field.Type), jsonName))
						}
					}
					structs[pkgTag+"."+decl.Name.Name] = fields
				case *ast.GenDecl:
					if decl.Tok != token.CONST {
						return true
					}
					for _, spec := range decl.Specs {
						valueSpec, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for i, name := range valueSpec.Names {
							if !strings.HasSuffix(name.Name, "PayloadType") {
								continue
							}
							if i < len(valueSpec.Values) {
								if lit, ok := valueSpec.Values[i].(*ast.BasicLit); ok {
									consts[name.Name] = lit.Value
								}
							}
						}
					}
				}
				return true
			})
		}
	}

	return nil
}

func renderWireFormat(structs map[string][]string, consts map[string]string) string {
	var b strings.Builder
	b.WriteString("# RPC wire format. Regenerate with: go test ./pkg/rpc -run TestWireFormatGolden -update-golden\n")

	b.WriteString("\n[payload types]\n")
	constNames := make([]string, 0, len(consts))
	for name := range consts {
		constNames = append(constNames, name)
	}
	sort.Strings(constNames)
	for _, name := range constNames {
		fmt.Fprintf(&b, "%s = %s\n", name, consts[name])
	}

	b.WriteString("\n[structs]\n")
	structNames := make([]string, 0, len(structs))
	for name := range structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)
	for _, name := range structNames {
		fmt.Fprintf(&b, "%s:\n", name)
		for _, field := range structs[name] {
			fmt.Fprintf(&b, "  %s\n", field)
		}
	}

	return b.String()
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// diffLines returns a compact line diff between two strings.
func diffLines(want string, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	wantSet := make(map[string]bool, len(wantLines))
	for _, l := range wantLines {
		wantSet[l] = true
	}
	gotSet := make(map[string]bool, len(gotLines))
	for _, l := range gotLines {
		gotSet[l] = true
	}

	var b strings.Builder
	for _, l := range wantLines {
		if !gotSet[l] {
			fmt.Fprintf(&b, "- %s\n", l)
		}
	}
	for _, l := range gotLines {
		if !wantSet[l] {
			fmt.Fprintf(&b, "+ %s\n", l)
		}
	}
	if b.Len() == 0 {
		return "(ordering difference)"
	}
	return b.String()
}
