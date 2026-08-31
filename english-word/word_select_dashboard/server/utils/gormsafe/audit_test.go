package gormsafe

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestProductionGormOpenUsesSecureConfig(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve audit test source path")
	}
	serverRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	requiredDirect := map[string]bool{
		"api/v1/system/sys_cloze_result.go":   false,
		"service/system/sys_initdb_mysql.go":  false,
		"service/system/sys_initdb_pgsql.go":  false,
		"service/system/sys_initdb_sqlite.go": false,
		"service/system/sys_initdb_mssql.go":  false,
	}

	err := filepath.WalkDir(serverRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		aliases := importAliases(parsed)
		relative, err := filepath.Rel(serverRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !isGormOpen(call.Fun, aliases) {
				return true
			}
			if len(call.Args) != 2 {
				t.Errorf("%s: gorm.Open must receive a secure config as its second argument", relative)
				return true
			}
			kind := secureConfigCallKind(call.Args[1], aliases)
			if kind == "" {
				t.Errorf("%s: production gorm.Open bypasses gormsafe.Config", relative)
				return true
			}
			if _, required := requiredDirect[relative]; required {
				if kind != "direct" {
					t.Errorf("%s: runtime/direct connection must call gormsafe.Config directly", relative)
					return true
				}
				requiredDirect[relative] = true
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("audit production GORM opens: %v", err)
	}
	for path, found := range requiredDirect {
		if !found {
			t.Errorf("%s: required direct gormsafe.Config connection was not found", path)
		}
	}
}

func importAliases(file *ast.File) map[string]string {
	aliases := make(map[string]string, len(file.Imports))
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		aliases[name] = path
	}
	return aliases
}

func isGormOpen(function ast.Expr, aliases map[string]string) bool {
	selector, ok := function.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Open" {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && aliases[identifier.Name] == "gorm.io/gorm"
}

func secureConfigCallKind(expression ast.Expr, aliases map[string]string) string {
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return ""
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Config" {
		return ""
	}
	if identifier, ok := selector.X.(*ast.Ident); ok && aliases[identifier.Name] == "github.com/conchi/go-react-template/server/utils/gormsafe" {
		return "direct"
	}
	internalGorm, ok := selector.X.(*ast.SelectorExpr)
	if !ok || internalGorm.Sel.Name != "Gorm" {
		return ""
	}
	identifier, ok := internalGorm.X.(*ast.Ident)
	if ok && aliases[identifier.Name] == "github.com/conchi/go-react-template/server/initialize/internal" {
		return "startup"
	}
	return ""
}
