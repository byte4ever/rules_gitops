// Binary fast_template_engine expands templates using
// stamp info files and explicit variable substitutions.
package main

import (
	"flag"
	"log"

	"github.com/byte4ever/rules_gitops/templating"
)

type arrayFlags []string

func (af *arrayFlags) String() string {
	return ""
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

func main() {
	var (
		stampInfoFile arrayFlags
		variable      arrayFlags
		imports       arrayFlags
		output        string
		tpl           string
		executable    bool
		startTag      string
		endTag        string
	)

	flag.Var(
		&stampInfoFile,
		"stamp_info_file",
		"Stamp info file path (repeatable)",
	)

	flag.Var(
		&variable,
		"variable",
		"Variable in NAME=VALUE format (repeatable)",
	)

	flag.Var(
		&imports,
		"imports",
		"Import in NAME=filename format (repeatable)",
	)

	flag.StringVar(
		&output, "output", "",
		"Output file path (stdout if empty)",
	)

	flag.StringVar(
		&tpl, "template", "",
		"Input template file path",
	)

	flag.BoolVar(
		&executable, "executable", false,
		"Set executable bit on output file",
	)

	flag.StringVar(
		&startTag, "start_tag", "{{",
		"Start tag for template placeholders",
	)

	flag.StringVar(
		&endTag, "end_tag", "}}",
		"End tag for template placeholders",
	)

	flag.Parse()

	en := templating.Engine{
		StartTag:       startTag,
		EndTag:         endTag,
		StampInfoFiles: stampInfoFile,
	}

	if err := en.Expand(
		tpl, output, variable, imports, executable,
	); err != nil {
		log.Fatal(err)
	}
}
