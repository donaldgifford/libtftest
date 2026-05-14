package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Marker is a single `libtftest:requires` entry harvested from source.
type Marker struct {
	Function string   `json:"function"`
	Receiver string   `json:"receiver,omitempty"`
	Package  string   `json:"package"`
	Tags     []string `json:"tags"`
	Reason   string   `json:"reason"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
}

// IR is the intermediate representation emitted by `scan` and consumed
// by `render`. The JSON shape is the public contract between the two
// commands.
type IR struct {
	Markers []Marker `json:"markers"`
}

// markerRE matches a single line of the form
//
//	// libtftest:requires <tag>[,<tag>...] <reason>
//
// Capture group 1 is the comma-separated tag list, group 2 is the
// reason (free-form, runs to end of line).
var markerRE = regexp.MustCompile(`^//\s*libtftest:requires\s+([a-z][a-z0-9_,-]*)\s+(.+?)\s*$`)

// scan walks root for .go files and returns every marker found,
// sorted in the canonical order (package, file, line) for stable
// output across runs.
func scan(root string) (*IR, error) {
	var markers []Marker

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
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		fileMarkers, err := scanFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		markers = append(markers, fileMarkers...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(markers, func(i, j int) bool {
		if markers[i].Package != markers[j].Package {
			return markers[i].Package < markers[j].Package
		}
		if markers[i].File != markers[j].File {
			return markers[i].File < markers[j].File
		}
		return markers[i].Line < markers[j].Line
	})

	return &IR{Markers: markers}, nil
}

// scanFile parses one Go file and returns every `libtftest:requires`
// marker paired with the function the comment is attached to.
func scanFile(path string) ([]Marker, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var out []Marker
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}

		for _, c := range fn.Doc.List {
			tags, reason, ok := parseMarker(c.Text)
			if !ok {
				continue
			}
			out = append(out, Marker{
				Function: fn.Name.Name,
				Receiver: receiverType(fn),
				Package:  file.Name.Name,
				Tags:     tags,
				Reason:   reason,
				File:     path,
				Line:     fset.Position(c.Pos()).Line,
			})
		}
	}
	return out, nil
}

// parseMarker returns the (tags, reason, ok) parsed from a single
// comment line. ok=false means the line is not a marker.
func parseMarker(line string) ([]string, string, bool) {
	m := markerRE.FindStringSubmatch(line)
	if m == nil {
		return nil, "", false
	}
	tags := strings.Split(m[1], ",")
	for i, t := range tags {
		tags[i] = strings.TrimSpace(t)
	}
	return tags, strings.TrimSpace(m[2]), true
}

// receiverType returns the receiver type name for a method declaration
// (the bare type name without pointer indirection), or "" for a
// package-level function.
func receiverType(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return ""
	}
	switch t := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

// shouldSkipDir reports whether a directory should be skipped during
// the walk. Skips VCS metadata, build outputs, vendor mirrors, the
// tool itself, and other directories that contain Go source we don't
// want to scan.
func shouldSkipDir(path string) bool {
	base := filepath.Base(path)
	switch base {
	case ".git", "node_modules", "vendor", "build", ".claude", ".docz":
		return true
	}
	return false
}
