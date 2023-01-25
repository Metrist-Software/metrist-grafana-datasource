//go:build mage
// +build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"

	"fmt"
	"os"

	"github.com/magefile/mage/sh"
)

// Generate a production build (pointing to the production API)
func BuildProd() {
	buildForEnv("prod")
}

// Generate a dev build (pointing to the dev API)
func BuildDev() {
	buildForEnv("dev")
}

// Generate a dev or production built dependent on the GITHUB_REF env var (defaults to dev)
func Build() {
	buildFunction := getBuildFunc()
	buildFunction()
}

// Default configures the default target.
var Default = Build

// Helper functions
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

func getBuildFunc() func() {
	environment, _ := os.LookupEnv("GITHUB_REF")
	if environment == "refs/heads/main" {
		return BuildProd
	} else {
		return BuildDev
	}
}
