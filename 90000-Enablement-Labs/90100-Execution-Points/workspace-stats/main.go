package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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

type SiloStats struct {
	Name       string
	TotalFiles int
	ExtCounts  map[string]int
}

func isSiloName(name string) bool {
	if len(name) < 5 {
		return false
	}

	// Check for nnnnn
	allDigits := true
	for i := 0; i < 5; i++ {
		if !unicode.IsDigit(rune(name[i])) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	// Check for Cnnnn
	if (name[0] == 'C' || name[0] == 'c') && len(name) >= 5 {
		allDigits = true
		for i := 1; i < 5; i++ {
			if !unicode.IsDigit(rune(name[i])) {
				allDigits = false
				break
			}
		}
		return allDigits
	}

	return false
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
	siloStatsMap := make(map[string]*SiloStats)

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

		// Update root-level silo stats
		if len(parts) > 0 {
			rootPart := parts[0]
			if isSiloName(rootPart) {
				if _, ok := siloStatsMap[rootPart]; !ok {
					siloStatsMap[rootPart] = &SiloStats{
						Name:      rootPart,
						ExtCounts: make(map[string]int),
					}
				}
				siloStatsMap[rootPart].TotalFiles++
				siloStatsMap[rootPart].ExtCounts[ext]++
			}
		}

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

	// Root Silo Summary
	if len(siloStatsMap) > 0 {
		fmt.Println("\n🏛️  Sovereign Silo Summary (nnnnn / Cnnnn):")
		var sortedSilos []string
		for k := range siloStatsMap {
			sortedSilos = append(sortedSilos, k)
		}
		sort.Strings(sortedSilos)

		for _, name := range sortedSilos {
			s := siloStatsMap[name]
			fmt.Printf("  ├── %s (%d files)\n", name, s.TotalFiles)
			
			// Show top 3 extensions for this silo
			type extCount struct {
				ext string
				val int
			}
			var siloExts []extCount
			for ext, count := range s.ExtCounts {
				siloExts = append(siloExts, extCount{ext, count})
			}
			sort.Slice(siloExts, func(i, j int) bool {
				return siloExts[i].val > siloExts[j].val
			})
			
			for i := 0; i < len(siloExts) && i < 3; i++ {
				fmt.Printf("  │   - %-12s : %d\n", siloExts[i].ext, siloExts[i].val)
			}
		}
	}

	fmt.Println("\n📂 Directory Tree (Recursive Counts):")
	printTree(rootStats, "", 0, *maxDepth)

	fmt.Println("\n📄 Global File Type Breakdown:")
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
