package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/log"

	"github.com/groq/kustomize-lint/internal" // For Version
	"github.com/groq/kustomize-lint/pkg/kustomization"
)

type DebugFlag bool

func (d DebugFlag) BeforeApply() error {
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	return nil
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println(vars["version"])
	app.Exit(0)
	return nil
}

type Globals struct {
	Version VersionFlag `name:"version" help:"Print version information and quit"`
	Debug   DebugFlag   `name:"debug" short:"d" help:"Enable debug mode"`
}

type CLI struct {
	Globals

	Lint LintCmd `cmd:"" help:"Lint kustomization files"`
}

type LintCmd struct {
	Path     string   `arg:"" name:"path" help:"Path to validate." type:"path"`
	Excludes []string `name:"exclude" short:"x" help:"Exclude files matching the given glob patterns."`
}

func (cmd *LintCmd) Run(globals *Globals) error {
	cmd.Excludes = append(cmd.Excludes, "README.md", ".gitignore")
	referenceLoader := kustomization.NewReferenceLoader(cmd.Excludes...)

	if err := referenceLoader.Validate(cmd.Path); err != nil {
		log.Fatal("Validation errors", "err", err)
	}

	return nil
}

func init() {
	log.SetReportTimestamp(false)
	log.SetReportCaller(false)
}

func main() {
	cli := CLI{}

	ctx := kong.Parse(&cli,
		kong.Name("kustomize-lint"),
		kong.Description("A linter for kustomization files"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": internal.Version,
		})
	ctx.FatalIfErrorf(ctx.Run(&cli.Globals))
}
