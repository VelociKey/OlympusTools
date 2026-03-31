package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Stats struct {
	TotalFiles int
	LocalFiles int
	SubDirs    map[string]*Stats
}

func newStats() *Stats {
	return &Stats{
		SubDirs: make(map[string]*Stats),
	}
}

type ExtensionCount struct {
	Ext   string
	Count int
}

func main() {
	workspace := flag.String("workspace", ".", "Path to scan")
	maxDepth := flag.Int("depth", 3, "Maximum tree depth")
	exclude := flag.String("exclude", ".git,node_modules", "Comma-separated directory names to ignore")
	flag.Parse()

	excludeMap := make(map[string]bool)
	for _, s := range strings.Split(*exclude, ",") {
		excludeMap[strings.TrimSpace(s)] = true
	}

	rootStats := newStats()
	extCounts := make(map[string]int)

	absRoot, err := filepath.Abs(*workspace)
	if err != nil {
		fmt.Printf("Error resolving path: %v\n", err)
		os.Exit(1)
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		if rel == "." {
			return nil
		}

		parts := strings.Split(rel, string(filepath.Separator))
		
		// Check exclusions
		for _, part := range parts {
			if excludeMap[part] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			return nil
		}

		// Update extension counts
		ext := filepath.Ext(path)
		if ext == "" {
			ext = "(no extension)"
		}
		extCounts[ext]++

		// Update tree stats
		curr := rootStats
		curr.TotalFiles++
		
		for i, part := range parts {
			if i >= *maxDepth {
				break
			}
			// If it's the last part and not a directory, it's a local file in the parent
			if i == len(parts)-1 {
				curr.LocalFiles++
				break
			}
			
			if _, ok := curr.SubDirs[part]; !ok {
				curr.SubDirs[part] = newStats()
			}
			curr = curr.SubDirs[part]
			curr.TotalFiles++
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n📊 Workspace Statistics: %s\n", absRoot)
	fmt.Println(strings.Repeat("-", 50))

	fmt.Println("\n📂 Directory Tree (Recursive Counts):")
	printTree(rootStats, "", 0, *maxDepth)

	fmt.Println("\n📄 File Type Breakdown:")
	var sortedExts []ExtensionCount
	for ext, count := range extCounts {
		sortedExts = append(sortedExts, ExtensionCount{ext, count})
	}
	sort.Slice(sortedExts, func(i, j int) bool {
		return sortedExts[i].Count > sortedExts[j].Count
	})

	for _, ec := range sortedExts {
		fmt.Printf("  %-15s : %d\n", ec.Ext, ec.Count)
	}

	fmt.Println("\n💡 Suggested Alternate Views:")
	fmt.Println("  - By Silo (00000-90000): Identify functional knowledge boundaries.")
	fmt.Println("  - By Lifecycle Stage (00SDLC-99PUBL): Analyze engineering process distribution.")
	fmt.Println("  - By Metadata Density: Locate areas with high governance (.jebnf concentration).")
}

func printTree(stats *Stats, name string, indent int, maxDepth int) {
	if indent > maxDepth {
		return
	}

	prefix := strings.Repeat("  ", indent)
	if name != "" {
		fmt.Printf("%s├── %s (%d total)\n", prefix, name, stats.TotalFiles)
	}

	keys := make([]string, 0, len(stats.SubDirs))
	for k := range stats.SubDirs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		printTree(stats.SubDirs[k], k, indent+1, maxDepth)
	}
}
