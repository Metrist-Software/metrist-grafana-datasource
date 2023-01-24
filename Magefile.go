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

// Generate a production build
func BuildProd() {
	buildForEnv("prod")
}

// Generate a normal development build
func Build() {
	buildForEnv("dev")
}

// Perform a production build and then upload to S3 distribution bucket
func Deploy() {
	qualifier, buildFunc := getQualifierAndBuildFunc()

	buildFunc()

	hash := getHash()
	distFileName := fmt.Sprintf("grafana-plugin-%s%s.zip", hash, qualifier)
	localFile := "/tmp/" + distFileName
	os.Remove(localFile)
	os.Chdir("dist")
	sh.Run("zip", "-r", localFile, ".")
	sh.Run("aws", "s3", "cp", "--region=us-west-2", localFile, "s3://dist.metrist.io/grafana-plugin/"+distFileName)
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

func getQualifierAndBuildFunc() (string, func()) {
	environment, _ := os.LookupEnv("GITHUB_REF")
	if environment == "refs/heads/main" {
		return "", BuildProd
	} else {
		return "-preview", Build
	}
}
