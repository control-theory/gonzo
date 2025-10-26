package main

import (
	"fmt"
	"os"
	"github.com/control-theory/gonzo/cmd"
	goversion "github.com/caarlos0/go-version"
	_ "embed"
)

// Build variables - set by ldflags during build
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	builtBy = "unknown"
)

const website = "https://www.controltheory.com/gonzo/"

//go:embed art.txt
var asciiArt string

func buildVersion(version, commit, buildTime, builtBy string) goversion.Info {
	return goversion.GetVersionInfo(
		goversion.WithAppDetails("gonzo", "Gonzo! The Go based TUI log analysis tool", website),
		goversion.WithASCIIName(asciiArt),
		func(i *goversion.Info) {
			if version != "" {
				i.GitVersion = version
			}
			if commit != "" {
				i.GitCommit = commit
			}
			if buildTime != "" {
				i.BuildDate = buildTime
			}
			if builtBy != "" {
				i.BuiltBy = builtBy
			}
		},
	)
}

func main() {
	version := buildVersion(version, commit, buildTime, builtBy)
	rootCmd := cmd.NewRootCmd(version)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
