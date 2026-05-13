package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "scan":
		err = scanCmd(args)
	case "render":
		err = renderCmd(args)
	case "check":
		err = checkCmd(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "docgen: unknown subcommand %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "docgen %s: %v\n", cmd, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `docgen — libtftest:requires marker tooling

Usage:
  docgen scan   [-root <dir>] [-out <path>]
  docgen render [-in <path>]  [-out <path>]
  docgen check  [-root <dir>]
  docgen help

Subcommands:
  scan    Walk the repo for libtftest:requires markers; emit JSON IR.
  render  Read JSON IR (stdin or -in); write Markdown to stdout or -out.
  check   Verify every libtftest.RequirePro caller has a marker.
`)
}

func scanCmd(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root to scan")
	out := fs.String("out", "", "output file (default stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ir, err := scan(*root)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(ir, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeOut(*out, data)
}

func renderCmd(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	in := fs.String("in", "", "input JSON IR (default stdin); if empty AND no piped data, run scan automatically")
	out := fs.String("out", "", "output file (default stdout)")
	root := fs.String("root", ".", "repo root to scan when -in is unset and no stdin is piped")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ir, err := loadIR(*in, *root)
	if err != nil {
		return err
	}
	return writeOut(*out, render(ir))
}

func checkCmd(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	root := fs.String("root", ".", "repo root to scan")
	if err := fs.Parse(args); err != nil {
		return err
	}

	violations, err := check(*root)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}
	for _, v := range violations {
		symbol := v.Function
		if v.Receiver != "" {
			symbol = v.Receiver + "." + v.Function
		}
		fmt.Fprintf(os.Stderr,
			"%s:%d  %s.%s  calls libtftest.RequirePro without a `// libtftest:requires` marker\n",
			v.File, v.Line, v.Package, symbol)
	}
	return fmt.Errorf("%d marker violation(s)", len(violations))
}

func loadIR(in, root string) (*IR, error) {
	switch {
	case in != "":
		data, err := os.ReadFile(in)
		if err != nil {
			return nil, err
		}
		return parseIR(data)
	case stdinPiped():
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		return parseIR(data)
	default:
		return scan(root)
	}
}

func parseIR(data []byte) (*IR, error) {
	var ir IR
	if err := json.Unmarshal(data, &ir); err != nil {
		return nil, err
	}
	return &ir, nil
}

func stdinPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

func writeOut(path string, data []byte) error {
	if path == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
}
