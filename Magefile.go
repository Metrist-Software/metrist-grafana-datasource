//go:build mage
// +build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"

	"fmt"

	"github.com/magefile/mage/sh"
)

// Generate a production build
func BuildProd() {
	buildForEnv("prod")
}

// Generate a normal development build
func Build() {
	buildForEnv("dev")
}

// Default configures the default target - will build dev by default.
var Default = Build

//Helper functions

func getHash() string {
	if hash, err := sh.Output("git", "rev-parse", "--short", "HEAD"); err != nil {
		fmt.Println(err)
		return ""
	} else {
		return hash
	}
}

func buildForEnv(env string) {
	// We are using Grafana's build functions so if we want custom ldflag values we have to hook here
	build.SetBeforeBuildCallback(func(cfg build.Config) (build.Config, error) {
		cfg.CustomVars = map[string]string{
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.Environment": env,
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.BuildHash":   getHash(),
		}

		return cfg, nil
	})

	build.BuildAll()
}
