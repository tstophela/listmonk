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

	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("version", false, "show current version and build information")
	f.Bool("new-config", false, "generate a new sample config.toml file")
	f.Bool("install", false, "install and initialize the database schema")
	f.Bool("upgrade", false, "upgrade the database schema to the latest version")
	f.Bool("yes", false, "assume 'yes' to prompts during install/upgrade")
	f.Bool("idempotent", false, "make --install run idempotently")
	f.Bool("static-dir", false, "path to override the embedded static directory")

	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error parsing flags: %v", err)
	}

	// Display version and exit.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Printf("listmonk version %s | build: %s\n", version, buildString)
		os.Exit(0)
	}

	// Load config files.
	cfgFiles, _ := f.GetStringSlice("config")
	for _, c := range cfgFiles {
		lo.Printf("loading config file: %s", c)
		if err := ko.Load(file.Provider(c), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				lo.Printf("config file not found, skipping: %s", c)
				continue
			}
			lo.Fatalf("error loading config file %s: %v", c, err)
		}
	}

	// Load environment variables (prefix LISTMONK_).
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading environment variables: %v", err)
	}

	// Load CLI flag overrides (highest priority).
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading flag config: %v", err)
	}
}

func main() {
	lo.Printf("starting listmonk version %s | build: %s", version, buildString)

	// Initialize the app.
	app := &App{
		log:    lo,
		config: ko,
	}

	_ = app

	// TODO: Initialize database, file system, and HTTP server.
	lo.Println("listmonk initialized successfully")
}
