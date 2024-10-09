package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"text/template"

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
// TODO: the full paths should be urls
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

type Post struct {
	Title   string
	Date    string
	Content string
}

// At the top of each markdown source, there will
// be a header that ends with a markdown line
func findHeaderSeparator(lines []string) int {
	offset := 0
	for lineIndex, line := range lines {
		isSeparator := true
		offset += len(line)

		for i := 0; i < len(line); i++ {
			if line[i] != '-' {
				isSeparator = false
				break
			}
		}

		if isSeparator {
			return lineIndex
		}
	}

	return -1
}

func parsePost(file []byte) (Post, error) {
	lines := strings.Split(string(file), "\n")
	lineIndex := findHeaderSeparator(lines)
	if lineIndex == -1 {
		return Post{}, errors.New("EXPECTING HEADER")
	}

	// Extract yaml like info the struct
	post := Post{}
	for _, line := range lines[:lineIndex] {
		parts := strings.Split(line, ":")
		key := strings.Trim(parts[0], " ")
		value := strings.Trim(parts[1], " ")
		if key == "Title" {
			post.Title = value
		} else if key == "Date" {
			post.Date = value
		}
	}

	// Get the remaining content
	offset := 0
	for _, line := range lines[:lineIndex+1] {
		offset += len(line)
	}
	offset += len(lines[lineIndex])
	post.Content = strings.Join(lines[lineIndex:], "\n")
	return post, nil
}

// Transpile markdown into html
// TODO: indentation
func transpileMarkdown(source []byte) []byte {
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)

	document := p.Parse(source)
	// TODO: move this out
	//document, err = replaceLocalFilePaths(document)
	//if err != nil {
	//	panic(err)
	//}

	opts := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(opts)
	return markdown.Render(document, renderer)
}

func main() {
	mdFile, err := os.ReadFile("home.md")
	if err != nil {
		panic(err)
	}

	templateFile, err := os.ReadFile("template.html")
	if err != nil {
		panic(err)
	}

	post, err := parsePost(mdFile)
	if err != nil {
		panic(err)
	}
	// TODO: make this less dodgy
	htmlOutput := transpileMarkdown([]byte(post.Content))
	post.Content = string(htmlOutput)

	builder := new(strings.Builder)
	t := template.New("Article")
	t = template.Must(t.Parse(string(templateFile)))
	t.Execute(builder, post)

	// Output html
	err = os.WriteFile("home.html", []byte(builder.String()), 0777)
	if err != nil {
		panic(err)
	}
}
