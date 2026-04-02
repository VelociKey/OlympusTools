// Package main provides a deterministic tool to find go.mod and go.work files.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	rootDir := flag.String("root", ".", "Root directory to start search")
	maxDepth := flag.Int("depth", 4, "Maximum search depth")
	flag.Parse()

	absRoot, err := filepath.Abs(*rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Calculate current depth relative to root
		rel, _ := filepath.Rel(absRoot, path)
		depth := strings.Count(rel, string(os.PathSeparator))
		if rel == "." {
			depth = 0
		}

		if d.IsDir() {
			// Skip known high-volume/non-source directories
			name := d.Name()
			if name == ".git" || name == "bazel-out" || name == "node_modules" || strings.HasPrefix(name, "@") || name == ".antigravity" {
				return filepath.SkipDir
			}
			// Skip if too deep
			if depth >= *maxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for target files
		name := d.Name()
		if name == "go.mod" || name == "go.work" {
			fmt.Println(path)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Walk error: %v\n", err)
		os.Exit(1)
	}
}
