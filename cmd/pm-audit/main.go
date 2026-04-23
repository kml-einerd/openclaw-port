// Command cycle detects Go import cycles in the PM-OS codebase.
//
// Adapted from openclaw/scripts/check-import-cycles.ts.
package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"./..."}
	}

	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedImports}
	pkgs, err := packages.Load(cfg, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to load packages: %v\n", err)
		os.Exit(1)
	}

	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	visited := make(map[string]bool)
	path := make([]string, 0)
	inPath := make(map[string]bool)
	cycleFound := false

	var dfs func(p *packages.Package)
	dfs = func(p *packages.Package) {
		if cycleFound {
			return
		}
		if inPath[p.ID] {
			fmt.Println("ERROR: Import cycle detected!")
			for _, node := range path {
				fmt.Printf(" -> %s\n", node)
			}
			fmt.Printf(" -> %s\n", p.ID)
			cycleFound = true
			return
		}
		if visited[p.ID] {
			return
		}

		visited[p.ID] = true
		inPath[p.ID] = true
		path = append(path, p.ID)

		for _, imp := range p.Imports {
			dfs(imp)
		}

		path = path[:len(path)-1]
		inPath[p.ID] = false
	}

	for _, p := range pkgs {
		dfs(p)
	}

	if cycleFound {
		os.Exit(1)
	}

	fmt.Println("No import cycles detected.")
}
