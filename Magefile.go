//go:build mage
// +build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"

	"os"

	"github.com/magefile/mage/sh"
)

// Generate a production build (pointing to the production API)
func BuildProd() error {
	return buildForEnv("prod")
}

// Generate a dev build (pointing to the dev API)
func BuildDev() error {
	return buildForEnv("dev")
}

// Generate a dev or production built dependent on the GITHUB_REF env var (defaults to dev)
func Build() error {
	buildFunction := getBuildFunc()
	return buildFunction()
}

// Default configures the default target.
var Default = Build

// Helper functions
func getHash() (hash string, err error) {
	hash, err = sh.Output("git", "rev-parse", "--short", "HEAD")
	return
}

func buildForEnv(env string) error {
	buildHash, err := getHash()

	if err != nil {
		return err
	}

	// We are using Grafana's build functions so if we want custom ldflag values we have to hook here
	if err := build.SetBeforeBuildCallback(func(cfg build.Config) (build.Config, error) {
		cfg.CustomVars = map[string]string{
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.Environment": env,
			"github.com/Metrist-Software/metrist-grafana-datasource/pkg/internal.BuildHash":   buildHash,
		}

		return cfg, nil
	}); err != nil {
		return err
	}

	build.BuildAll()
	return nil
}

func getBuildFunc() func() error {
	environment, _ := os.LookupEnv("GITHUB_REF")
	if environment == "refs/heads/main" {
		return BuildProd
	}

	return BuildDev
}
