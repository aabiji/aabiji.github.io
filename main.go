package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func inDirectory(directory, file string) bool {
	files, err := os.ReadDir(directory)
	if err != nil {
		return false
	}

	for _, entry := range files {
		if !entry.IsDir() && entry.Name() == file {
			return true
		}
	}

	return false
}

// Get the full path to local file paths
func getFullPath(path string) ([]byte, error) {
	pathParts := strings.Split(path, ".")
	extension := pathParts[len(pathParts)-1]

	// It's a link to another markdown file, check if we can
	// find it in the sources folder
	if extension == "md" && inDirectory("src/", path) {
		return []byte("src/" + path), nil
	} else if extension == "md" {
		return nil, fmt.Errorf("%s is not found in src/", path)
	}

	// At this point it should be an asset
	if !inDirectory("assets/", path) {
		return []byte("assets/" + path), nil
	}
	return nil, fmt.Errorf("%s is not found in src/", path)
}

func replaceLocalFilePaths(document ast.Node) (ast.Node, error) {
	var buildError error
	ast.WalkFunc(document, func(node ast.Node, entering bool) ast.WalkStatus {
		if link, ok := node.(*ast.Link); ok && entering {
			path := string(link.Destination)
			_, err := url.ParseRequestURI(path)
			if err == nil {
				return ast.GoToNext
			}

			fullpath, err := getFullPath(path)
			if err != nil {
				buildError = err
				return ast.Terminate
			}
			link.Destination = fullpath
		}
		return ast.GoToNext
	})
	return document, buildError
}

func main() {
	filename := "test.md"
	file, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	document := p.Parse(file)
	document, err = replaceLocalFilePaths(document)
	if err != nil {
		panic(err)
	}

	flags := html.CommonFlags
	opts := html.RendererOptions{Flags: flags}
	renderer := html.NewRenderer(opts)

	htmlStr := markdown.Render(document, renderer)
	fmt.Println(htmlStr)
}
