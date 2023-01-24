//go:build mage
// +build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"

	"github.com/magefile/mage/sh"
)

func BuildProd() {
	build.SetBeforeBuildCallback(func(cfg build.Config) (build.Config, error) {
		cfg.CustomVars = map[string]string{
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.Environment": "prod",
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.Hash":        hash(),
		}

		return cfg, nil
	})

	build.BuildAll()
}

// Default configures the default target.
var Default = build.BuildAll

func hash() string {
	hash, _ := sh.Output("git", "rev-parse", "--short", "HEAD")
	return hash
}
