package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func readFile(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func onlyContains(str string, char byte) bool {
	for i := 0; i < len(str); i++ {
		if str[i] != char {
			return false
		}
	}
	return true
}

type PostHeader struct {
	Info     map[string]string
	endIndex int // Index where the post header ends
}

func parsePostHeader(source string) (PostHeader, error) {
	header := PostHeader{Info: make(map[string]string, 1)}
	lines := strings.Split(source, "\n")

	for _, line := range lines {
		// Ignore empty lines
		if len(strings.Trim(line, " ")) == 0 {
			continue
		}

		// A post header ends with a horizantal line separator
		header.endIndex += len(line) + 1
		if onlyContains(line, '-') {
			break
		}

		pair := strings.Split(line, ":")
		if len(pair) != 2 {
			return PostHeader{}, errors.New("INVALID KEY VALUE")
		}

		key := strings.Trim(pair[0], " ")
		key = strings.ToLower(key)
		value := strings.Trim(pair[1], " ")
		header.Info[key] = value
	}

	return header, nil
}

func realFilePath(path, sourceFolder, outputBase string) (string, error) {
	files, err := os.ReadDir(sourceFolder)
	if err != nil {
		return "", err
	}

	for _, entry := range files {
		if !entry.IsDir() && entry.Name() == path {
			// Found the path inside the source folder,
			// return the new file path
			return fmt.Sprintf("%s/%s", outputBase, path), nil
		}
	}

	return "", fmt.Errorf("%s NOT FOUND IN %s", path, sourceFolder)
}

// TODO: test this
// TODO: do we even need this?? How are assets actually sourced on github pages???
func replaceLocalFilePaths(document ast.Node) (ast.Node, error) {
	var buildError error
	outputBase := "https://aabiji.github.com"
	sourceFolder := "src/"
	assetFolder := "assets/"

	ast.WalkFunc(document, func(node ast.Node, entering bool) ast.WalkStatus {
		if link, ok := node.(*ast.Link); ok && entering {
			dest := string(link.Destination)

			// Ignore valid urls
			_, err := url.ParseRequestURI(dest)
			if err == nil || len(dest) == 0 {
				return ast.GoToNext
			}

			path := ""
			pathParts := strings.Split(dest, ".")
			extension := pathParts[len(pathParts)-1]
			if extension == "md" {
				// Fix links to other markdown files (blog posts)
				path, buildError = realFilePath(dest, sourceFolder, outputBase)
			} else {
				// Fix links to potential asset files
				path, buildError = realFilePath(dest, assetFolder, outputBase)
			}

			if buildError != nil {
				return ast.Terminate
			}
			link.Destination = []byte(path)
		}
		return ast.GoToNext
	})

	return document, buildError
}

func transpileMarkdown(source string) ([]byte, error) {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	document := p.Parse([]byte(source))

	// Apply AST transformations
	document, err := replaceLocalFilePaths(document)
	if err != nil {
		return nil, err
	}

	options := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(options)
	return markdown.Render(document, renderer), nil
}

func main() {
	source, err := readFile("home.md")
	if err != nil {
		panic(err)
	}
	header, err := parsePostHeader(source)
	if err != nil {
		panic(err)
	}
	htmlSource, err := transpileMarkdown(source[header.endIndex:])
	if err != nil {
		panic(err)
	}
	fmt.Println(string(htmlSource))
	// TODO: make this less dodgy
	//htmlOutput := transpileMarkdown([]byte(post.Content))
	//post.Content = string(htmlOutput)
	//
	//builder := new(strings.Builder)
	//t := template.New("Article")
	//t = template.Must(t.Parse(string(templateFile)))
	//t.Execute(builder, post)
	//
	//// Output html
	//err = os.WriteFile("home.html", []byte(builder.String()), 0777)
	//if err != nil {
	//	panic(err)
	//}
}
