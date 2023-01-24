//go:build mage
// +build mage

package main

import (
	"os"

	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"
)

func init() {
	// use the default if not set
	environment, ok := os.LookupEnv("ENV")
	if !ok {
		environment = "dev"
	}
	build.SetBeforeBuildCallback(func(cfg build.Config) (build.Config, error) {
		cfg.CustomVars = map[string]string{
			"internal.Environment": environment,
		}

		return cfg, nil
	})
}

// Default configures the default target.
var Default = build.BuildAll
