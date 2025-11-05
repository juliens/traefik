package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

type Dependency struct {
	Module  string
	Version string
	Replace string // The actual version used if there was a replace
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <reference-version-file> <target-version-file>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Compares two 'go version -m' outputs and identifies dependencies\n")
		fmt.Fprintf(os.Stderr, "that exist in both but have different versions.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  go version -m ./traefik > ref.txt\n")
		fmt.Fprintf(os.Stderr, "  go version -m ./plugin.so > target.txt\n")
		fmt.Fprintf(os.Stderr, "  %s ref.txt target.txt\n", os.Args[0])
		os.Exit(1)
	}

	referenceFile := args[0]
	targetFile := args[1]

	// Parse both files
	referenceDeps, err := parseDependencies(referenceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing reference file: %v\n", err)
		os.Exit(1)
	}

	targetDeps, err := parseDependencies(targetFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing target file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Reference file: %s (%d dependencies)\n", referenceFile, len(referenceDeps))
	fmt.Printf("Target file: %s (%d dependencies)\n", targetFile, len(targetDeps))
	fmt.Println()

	// Find mismatches
	mismatches := findMismatches(referenceDeps, targetDeps)

	if len(mismatches) == 0 {
		fmt.Println("No version mismatches found. All common dependencies are aligned.")
		return
	}

	fmt.Printf("Found %d mismatched dependencies:\n\n", len(mismatches))

	for _, mismatch := range mismatches {
		fmt.Printf("Module: %s\n", mismatch.Module)
		fmt.Printf("  Reference: %s\n", mismatch.ReferenceVersion)
		fmt.Printf("  Target:    %s\n", mismatch.TargetVersion)
		fmt.Println()
	}

	// Output go.mod replace directives
	fmt.Println("Suggested go.mod replace directives:")
	fmt.Println()
	for _, mismatch := range mismatches {
		fmt.Printf("replace %s => %s %s\n", mismatch.Module, mismatch.Module, mismatch.ReferenceVersion)
	}
}

type Mismatch struct {
	Module           string
	ReferenceVersion string
	TargetVersion    string
}

func parseDependencies(filename string) (map[string]Dependency, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	deps := make(map[string]Dependency)
	replaces := make(map[string]string) // module -> replaced version
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Parse dep lines: "	dep	module version"
		if strings.HasPrefix(line, "dep\t") || strings.HasPrefix(line, "dep ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				module := parts[1]
				version := parts[2]
				deps[module] = Dependency{
					Module:  module,
					Version: version,
				}
			}
		}

		// Parse replace lines: "	=>	module version"
		// These follow dep lines and indicate the actual version used
		if strings.HasPrefix(line, "=>\t") || strings.HasPrefix(line, "=> ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				replacedModule := parts[1]
				replacedVersion := parts[2]
				replaces[replacedModule] = replacedVersion
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Apply replaces to dependencies
	for module, dep := range deps {
		if replacedVersion, ok := replaces[module]; ok {
			dep.Replace = replacedVersion
			deps[module] = dep
		}
	}

	return deps, nil
}

func findMismatches(reference, target map[string]Dependency) []Mismatch {
	var mismatches []Mismatch

	for module, targetDep := range target {
		refDep, existsInRef := reference[module]
		if !existsInRef {
			// Module only in target, not a mismatch
			continue
		}

		// Get effective versions (considering replaces)
		refVersion := refDep.Version
		if refDep.Replace != "" {
			refVersion = refDep.Replace
		}

		targetVersion := targetDep.Version
		if targetDep.Replace != "" {
			targetVersion = targetDep.Replace
		}

		// Compare effective versions
		if refVersion != targetVersion {
			mismatches = append(mismatches, Mismatch{
				Module:           module,
				ReferenceVersion: refVersion,
				TargetVersion:    targetVersion,
			})
		}
	}

	return mismatches
}