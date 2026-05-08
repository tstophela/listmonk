package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	flag "github.com/spf13/pflag"
)

const (
	// Version of the application.
	version = "2.5.1"

	// Build info populated at compile time via ldflags.
	buildString = "unknown"
)

// App holds the global application state.
type App struct {
	log    *log.Logger
	fs     stuffbin.FileSystem
	config *koanf.Koanf
}

var (
	lo = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	ko = koanf.New(".")
)

func init() {
	// Register CLI flags.
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Default config paths: check both config.toml and config.toml.local so that
	// a local override file can be used without modifying the main config.
	// Also include config.toml.personal for personal overrides (my addition).
	f.StringSlice("config", []string{"config.toml", "config.toml.local", "config.toml.personal"},
		"path to one or more config files (will be merged in order)")
	f.Bool("version", false, "show current version and build information")
	f.Bool("new-config", false, "generate a new sample config.toml file")
	f.Bool("install", false, "install and initialize the database schema")
	f.Bool("upgrade", false, "upgrade the database schema to the latest version")
	f.Bool("yes", false, "assume 'yes' to prompts during install/upgrade")
	f.Bool("idempotent", false, "make --install run idempotently")
	// Note: --static-dir should be a string flag, not bool, to accept a directory path.
	f.String("static-dir", "", "path to override the embedded static directory")

	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error parsing flags: %v", err)
	}

	// Display version and exit.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Printf("listmonk version %s | build: %s\n", version, buildString)
		os.Exit(0)
	}

	// Load config files. Missing files are skipped silently; only unexpected
	// errors (e.g. parse errors) are treated as fatal.
	cfgFiles, _ := f.GetStringSlice("config")
	for _, c := range cfgFiles {
		if err := ko.Load(file.Provider(c), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				// Not logging skipped files to keep startup output clean.
				continue
			}
			lo.Fatalf("error loading config file %s: %v", c, err)
		}
		lo.Printf("loaded config file: %s", c)
	}

	// Load environment variables (prefix LISTMONK_).
	// Using "__" as the nested key separator so that e.g. LISTMONK_DB__HOST maps to db.host.
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading environment variables: %v", err)
	}

	// Load CLI flag overrides (highest priority).
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading flag overrides: %v", err)
	}
}
