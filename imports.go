package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

var (
	red  = color.New(color.FgHiRed).SprintFunc()
	blue = color.New(color.FgHiBlue).SprintFunc()
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	dir := flag.String("d", ".", "directory to search from")
	out := flag.String("o", "text", "output format [text|json|yaml]")
	flag.Parse()

	dirs := []string{}
	err := filepath.Walk(*dir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() && !strings.Contains(path, "vendor") {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("error walking directories: %v", red(err))
	}

	imports, err := getImports(dirs, srcDir())
	if len(imports) > 0 {
		print(imports, *out)
	}
}

func getImports(dirs []string, gopath string) (imports map[string][]string, err error) {
	imports = map[string][]string{}

	var pkgImports []string
	for _, dir := range dirs {
		fs := token.NewFileSet()
		nodes, err := parser.ParseDir(fs, dir, nil, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, node := range nodes {
			ast.Inspect(node, func(n ast.Node) bool {
				imp, ok := n.(*ast.ImportSpec)
				if ok {
					pkgImports = append(pkgImports, strings.Trim(imp.Path.Value, `"`))
					return true
				}
				return true
			})
		}

		if len(pkgImports) > 0 {
			fullPath, err := filepath.Abs(dir)
			if err != nil {
				return nil, errors.Wrap(err, "getting full path")
			}
			fullPath = strings.TrimPrefix(fullPath, gopath)
			imports[fullPath] = pkgImports
		}
		pkgImports = []string{}
	}

	return
}

func print(imports map[string][]string, out string) {
	switch strings.ToLower(out) {
	case "text":
		for k, v := range imports {
			fmt.Println(k, blue(v))
		}
	case "json":
		b, err := json.MarshalIndent(imports, "", "    ")
		if err != nil {
			log.Fatalf("error marshalling: %v", red(err))
		}
		fmt.Println(string(b))
	case "yaml":
		b, err := yaml.Marshal(imports)
		if err != nil {
			log.Fatalf("error marshalling: %v", red(err))
		}
		fmt.Println(string(b))
	}
}

func srcDir() string {
	p := os.Getenv("GOPATH")
	if p == "" {
		p = build.Default.GOPATH
	}

	return filepath.Join(p, "src") + "/"
}
