package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Violation is a single function found to call libtftest.RequirePro
// without an accompanying marker. Returned by check so callers can
// format the failure however they need; the default CLI renders
// them as `file:line  function  reason`.
type Violation struct {
	Function string
	Receiver string
	Package  string
	File     string
	Line     int
}

// check walks the repo and returns every Violation: a function that
// calls libtftest.RequirePro without a `// libtftest:requires`
// marker on its doc comment.
//
// Functions that have a marker but don't actually call RequirePro are
// permitted — markers may anticipate future gates or document
// transitively-pro behavior the static check can't see.
func check(root string) ([]Violation, error) {
	var violations []Violation

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileViolations, err := checkFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		violations = append(violations, fileViolations...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return violations[i].Line < violations[j].Line
	})
	return violations, nil
}

// checkFile inspects a single file for RequirePro-without-marker
// violations.
func checkFile(path string) ([]Violation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var out []Violation
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !callsRequirePro(fn) {
			continue
		}
		if hasMarker(fn) {
			continue
		}
		out = append(out, Violation{
			Function: fn.Name.Name,
			Receiver: receiverType(fn),
			Package:  file.Name.Name,
			File:     path,
			Line:     fset.Position(fn.Pos()).Line,
		})
	}
	return out, nil
}

// callsRequirePro reports whether fn contains a call to
// libtftest.RequirePro (either by qualified name or as a bare
// identifier when the call site is inside the libtftest package
// itself).
func callsRequirePro(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch f := call.Fun.(type) {
		case *ast.SelectorExpr:
			pkg, ok := f.X.(*ast.Ident)
			if ok && pkg.Name == "libtftest" && f.Sel.Name == "RequirePro" {
				found = true
				return false
			}
		case *ast.Ident:
			if f.Name == "RequirePro" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// hasMarker reports whether fn's doc comment contains at least one
// `// libtftest:requires` line.
func hasMarker(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}
	for _, c := range fn.Doc.List {
		if _, _, ok := parseMarker(c.Text); ok {
			return true
		}
	}
	return false
}
