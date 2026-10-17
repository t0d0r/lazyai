package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"
)

// flagDefs wraps flag.FlagSet with GNU-style options: every option has a
// long name (--json) and may have a short one (-j), and the help text is
// generated from the definitions.
type flagDefs struct {
	fs   *flag.FlagSet
	opts []optDoc
}

type optDoc struct {
	short, long, arg, def, help string
}

func newFlagDefs(name string) *flagDefs {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors and help are printed by run
	return &flagDefs{fs: fs}
}

func (f *flagDefs) Bool(short, long, help string) *bool {
	p := new(bool)
	f.fs.BoolVar(p, long, false, help)
	if short != "" {
		f.fs.BoolVar(p, short, false, help)
	}
	f.opts = append(f.opts, optDoc{short: short, long: long, help: help})
	return p
}

func (f *flagDefs) String(short, long, arg, def, help string) *string {
	p := new(string)
	f.fs.StringVar(p, long, def, help)
	if short != "" {
		f.fs.StringVar(p, short, def, help)
	}
	f.opts = append(f.opts, optDoc{short: short, long: long, arg: arg, def: def, help: help})
	return p
}

func (f *flagDefs) Duration(short, long, arg string, def time.Duration, help string) *time.Duration {
	p := new(time.Duration)
	f.fs.DurationVar(p, long, def, help)
	if short != "" {
		f.fs.DurationVar(p, short, def, help)
	}
	f.opts = append(f.opts, optDoc{short: short, long: long, arg: arg, def: def.String(), help: help})
	return p
}

// parse returns the positional arguments. With permute, options may appear
// anywhere before a "--"; without it, parsing stops at the first argument
// that is not an option, so the rest can be handed on untouched.
func (f *flagDefs) parse(args []string, permute bool) ([]string, error) {
	var tail []string
	if permute {
		for i, a := range args {
			if a == "--" {
				args, tail = args[:i], args[i+1:]
				break
			}
		}
	}
	var pos []string
	for {
		if err := f.fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError(err.Error())
		}
		args = f.fs.Args()
		if !permute || len(args) == 0 {
			return append(append(pos, args...), tail...), nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func (f *flagDefs) writeOptions(w io.Writer) {
	fmt.Fprintln(w, "Options:")
	docs := append(f.opts, optDoc{short: "h", long: "help", help: "show this help and exit"})
	for _, o := range docs {
		left := "    "
		if o.short != "" {
			left = "-" + o.short + ", "
		}
		left += "--" + o.long
		if o.arg != "" {
			left += " " + o.arg
		}
		help := o.help
		if o.def != "" {
			help += " (default " + o.def + ")"
		}
		fmt.Fprintf(w, "  %-24s %s\n", left, help)
	}
}

// usageError is a mistake in how the command was called; it exits with 2.
type usageError string

func (e usageError) Error() string { return string(e) }

func usagef(format string, a ...any) error {
	return usageError(fmt.Sprintf(format, a...))
}
