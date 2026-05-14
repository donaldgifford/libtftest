package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestParseMarker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		line   string
		tags   []string
		reason string
		ok     bool
	}{
		{
			name:   "single tag",
			line:   "// libtftest:requires pro LocalStack Pro for KMS GetKey",
			tags:   []string{"pro"},
			reason: "LocalStack Pro for KMS GetKey",
			ok:     true,
		},
		{
			name:   "multi-tag",
			line:   "// libtftest:requires pro,mockta Pro for IAM + mockta for Okta principals",
			tags:   []string{"pro", "mockta"},
			reason: "Pro for IAM + mockta for Okta principals",
			ok:     true,
		},
		{
			name:   "extra whitespace tolerated",
			line:   "//  libtftest:requires  pro  reason text  ",
			tags:   []string{"pro"},
			reason: "reason text",
			ok:     true,
		},
		{
			name: "not a marker",
			line: "// just a doc comment",
			ok:   false,
		},
		{
			name: "marker without reason",
			line: "// libtftest:requires pro",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tags, reason, ok := parseMarker(tt.line)
			if ok != tt.ok {
				t.Fatalf("parseMarker(%q) ok = %v, want %v", tt.line, ok, tt.ok)
			}
			if !ok {
				return
			}
			if !reflect.DeepEqual(tags, tt.tags) {
				t.Errorf("tags = %v, want %v", tags, tt.tags)
			}
			if reason != tt.reason {
				t.Errorf("reason = %q, want %q", reason, tt.reason)
			}
		})
	}
}

func TestScan_SingleAndMultiTag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "pkg", "single.go"), `package pkg

// SingleTag does something Pro-only.
//
// libtftest:requires pro LocalStack Pro for the GetX API
func SingleTag() {}
`)
	mustWrite(t, filepath.Join(dir, "pkg", "multi.go"), `package pkg

// MultiTag needs Pro + mockta.
//
// libtftest:requires pro,mockta both gates apply
func MultiTag() {}
`)

	ir, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Markers) != 2 {
		t.Fatalf("scan returned %d markers, want 2: %+v", len(ir.Markers), ir.Markers)
	}

	byName := map[string]Marker{}
	for _, m := range ir.Markers {
		byName[m.Function] = m
	}

	single := byName["SingleTag"]
	if !reflect.DeepEqual(single.Tags, []string{"pro"}) {
		t.Errorf("SingleTag.Tags = %v, want [pro]", single.Tags)
	}
	if single.Reason != "LocalStack Pro for the GetX API" {
		t.Errorf("SingleTag.Reason = %q", single.Reason)
	}

	multi := byName["MultiTag"]
	if !reflect.DeepEqual(multi.Tags, []string{"pro", "mockta"}) {
		t.Errorf("MultiTag.Tags = %v, want [pro mockta]", multi.Tags)
	}
}

func TestScan_StableOrder(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "z.go"), `package z

// libtftest:requires pro z reason
func Z() {}
`)
	mustWrite(t, filepath.Join(dir, "a.go"), `package a

// libtftest:requires pro a reason
func A() {}
`)

	ir, err := scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ir.Markers) != 2 {
		t.Fatalf("want 2 markers, got %d", len(ir.Markers))
	}
	// Package sort puts `a` before `z`.
	if ir.Markers[0].Package != "a" || ir.Markers[1].Package != "z" {
		t.Errorf("packages not sorted: %v", []string{ir.Markers[0].Package, ir.Markers[1].Package})
	}
}

func TestCheck_MissingMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "bad.go"), `package x

func Bad() {
	libtftest.RequirePro(nil)
}
`)

	vs, err := check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("want 1 violation, got %d: %+v", len(vs), vs)
	}
	if vs[0].Function != "Bad" {
		t.Errorf("violation func = %q, want Bad", vs[0].Function)
	}
}

func TestCheck_MarkerPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "good.go"), `package x

// Good is correctly marked.
//
// libtftest:requires pro reason
func Good() {
	libtftest.RequirePro(nil)
}
`)

	vs, err := check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Errorf("want 0 violations, got %d: %+v", len(vs), vs)
	}
}

func TestCheck_MarkerButNoRequireProIsAllowed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "anticipate.go"), `package x

// Anticipated future gate; marker exists but no RequirePro yet.
//
// libtftest:requires pro will gate this soon
func Anticipated() {}
`)

	vs, err := check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Errorf("marker without RequirePro should be allowed, got %d violations", len(vs))
	}
}

func TestCheck_TestFilesIgnored(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "x_test.go"), `package x

func TestSomething(t *testing.T) {
	libtftest.RequirePro(t)
}
`)

	vs, err := check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Errorf("test files must be skipped by check, got %d violations", len(vs))
	}
}

func TestRender_EmptyIR(t *testing.T) {
	t.Parallel()

	out := render(&IR{})
	if !strings.Contains(string(out), "_No markers found._") {
		t.Errorf("render on empty IR missing the no-markers banner; got:\n%s", string(out))
	}
}

func TestRender_TableShape(t *testing.T) {
	t.Parallel()

	ir := &IR{Markers: []Marker{
		{
			Function: "A",
			Package:  "x",
			Tags:     []string{"pro"},
			Reason:   "reason A",
			File:     "x/x.go",
			Line:     10,
		},
		{
			Function: "B",
			Receiver: "ABs",
			Package:  "x",
			Tags:     []string{"pro", "mockta"},
			Reason:   "reason | with pipe",
			File:     "x/x.go",
			Line:     20,
		},
	}}

	out := string(render(ir))

	if !strings.Contains(out, "| Symbol | Package | Tags | Reason | Source |") {
		t.Errorf("table header missing in:\n%s", out)
	}
	if !strings.Contains(out, "`ABs.B`") {
		t.Errorf("method receiver should render as `Receiver.Method`; got:\n%s", out)
	}
	if !strings.Contains(out, `reason \| with pipe`) {
		t.Errorf("pipe characters must be escaped in the Reason column; got:\n%s", out)
	}
	// Multi-tag rendering: both tokens present, sorted lexicographically.
	if !strings.Contains(out, "**mockta** **pro**") {
		t.Errorf("multi-tag rendering should sort tags; got:\n%s", out)
	}
}

func TestRender_Deterministic(t *testing.T) {
	t.Parallel()

	ir := &IR{Markers: []Marker{
		{Function: "A", Package: "x", Tags: []string{"pro"}, Reason: "r1", File: "x.go", Line: 1},
		{Function: "B", Package: "y", Tags: []string{"mockta"}, Reason: "r2", File: "y.go", Line: 5},
	}}
	first := render(ir)
	second := render(ir)
	if !bytes.Equal(first, second) {
		t.Error("render not deterministic on identical IR")
	}
}
