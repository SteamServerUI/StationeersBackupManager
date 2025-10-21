// build/build.go
//go:build ignore
// +build ignore

// run from root with `go run build/build.go`
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// ANSI color codes for styling terminal output
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

func main() {
	fmt.Printf("%s=== Starting Build Pipeline ===%s\n", colorCyan, colorReset)
	// Platforms to build for
	platforms := []struct {
		os   string
		arch string
	}{
		//{"windows", "amd64"},
		{"linux", "amd64"},
	}

	// Build for each platform
	for _, platform := range platforms {
		fmt.Printf("%s\nBuilding for %s/%s...%s\n", colorBlue, platform.os, platform.arch, colorReset)

		// Set OS and architecture for cross-compilation
		os.Setenv("GOOS", platform.os)
		os.Setenv("GOARCH", platform.arch)

		// Prepare the output file name with the new version, branch, and platform
		var outputName = "StationeersBackupManager"

		// Append appropriate extension based on platform
		if platform.os == "windows" {
			outputName += ".exe"
		}
		if platform.os == "linux" {
			outputName += ".x86_64"
		}

		// Output to /build
		outputPath := filepath.Join("./", outputName)

		// Run the go build command targeting mian.go at root
		cmd := exec.Command("go", "build", "-ldflags=-s -w", "-gcflags=-l=4", "-o", outputPath, "main.go")

		// Capture any output or errors
		cmdOutput, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("%s✗ Build failed for %s/%s:%s %s\nOutput: %s\n",
				colorRed, platform.os, platform.arch, colorReset, err, string(cmdOutput))
			log.Fatalf("Build process terminated")
		}

		fmt.Printf("%s✓ Build successful!%s Created: %s%s%s\n",
			colorGreen, colorReset, colorYellow, outputPath, colorReset)
	}

	fmt.Printf("%s\n=== Build Pipeline Completed ===%s\n", colorCyan, colorReset)
}
