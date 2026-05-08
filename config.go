package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	spf13flag "github.com/spf13/pflag"
)

// initConfig loads configuration from config file, environment variables,
// and command-line flags, in that order of precedence.
func initConfig(ko *koanf.Koanf, flags *spf13flag.FlagSet) error {
	// Load config from file(s) specified via --config flag.
	cfgFiles, _ := flags.GetStringSlice("config")
	for _, f := range cfgFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", f)
		}
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			return fmt.Errorf("error loading config from file %s: %w", f, err)
		}
	}

	// Load environment variables with the prefix LISTMONK_.
	// LISTMONK_app__address becomes app.address in the config.
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")),
			"__", ".", -1)
	}), nil); err != nil {
		return fmt.Errorf("error loading config from env: %w", err)
	}

	// Load command-line flags, which take the highest precedence.
	if err := ko.Load(posflag.Provider(flags, ".", ko), nil); err != nil {
		return fmt.Errorf("error loading config from flags: %w", err)
	}

	return nil
}

// registerFlags registers all command-line flags for the application.
func registerFlags(f *spf13flag.FlagSet) {
	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("install", false,
		"run first-time installation wizard")
	f.Bool("upgrade", false,
		"upgrade database schema to the latest version")
	f.Bool("version", false,
		"show current version and build information")
	f.Bool("new-config", false,
		"generate a new sample config.toml file")
	f.String("static-dir", "",
		"path to the directory with static files (overrides the embedded files)")
	f.String("i18n-dir", "",
		"path to the directory with i18n language files (overrides the embedded files)")
	f.Bool("yes", false,
		"assume 'yes' to prompts during --install/--upgrade")
	f.Bool("idempotent", false,
		"make --install idempotent (do not fail if already installed)")
}
