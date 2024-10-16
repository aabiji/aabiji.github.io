// A tiny static site generator -- it basically just
// transpiles all the markdown files in the posts folder
// to html and outputs them to the html folder. It's really basic,
// but it works well for my purposes. The workflow involves
// writing a markdown file in the posts folder, then running
// `go run .` to build the site.
// TODO: maybe be smarter and automatically delete unused files

package main

import (
	"errors"
	"fmt"
	"html/template"
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

type Post struct {
	Content     template.HTML
	Info        map[string]string
	StylesPath  string
	headerIndex int // Index where the post header ends
}

func parsePostHeader(source string) (Post, error) {
	post := Post{Info: make(map[string]string, 1)}
	lines := strings.Split(source, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 { // Ignore empty lines
			continue
		}

		// A post header ends with a horizantal line separator
		post.headerIndex += len(line) + 1
		if onlyContains(line, '-') {
			break
		}

		pair := strings.Split(line, ":")
		if len(pair) != 2 {
			return Post{}, errors.New("INVALID KEY VALUE")
		}

		key := strings.Trim(pair[0], " ")
		key = strings.ToLower(key)
		value := strings.Trim(pair[1], " ")
		post.Info[key] = value
	}

	return post, nil
}

func indentHtml(htmlOutput []byte, numSpaces int) string {
	output := ""
	indent := strings.Repeat(" ", numSpaces)
	lines := strings.Split(string(htmlOutput), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue // ignore empty lines
		}
		if i != 0 { // The first line should already be indented
			output += indent
		}
		output += line + "\n"
	}
	return output
}

func fixRelativeFilPaths(document ast.Node) ast.Node {
	ast.WalkFunc(document, func(node ast.Node, entering bool) ast.WalkStatus {
		link, isLink := node.(*ast.Link)
		img, isImage := node.(*ast.Image)
		if (!isLink && !isImage) || !entering {
			return ast.GoToNext
		}

		dest := ""
		if isLink {
			dest = string(link.Destination)
		} else {
			dest = string(img.Destination)
		}

		// Ignore valid urls
		_, err := url.ParseRequestURI(dest)
		if err == nil || len(dest) == 0 {
			return ast.GoToNext
		}

		// Links to markdown files should really point to the corresponding html files
		pathParts := strings.Split(dest, ".")
		extension := pathParts[len(pathParts)-1]
		if extension == "md" {
			dest = fmt.Sprintf("html/%s.html", pathParts[0])
		} else {
			dest = "https://aabiji.github.io/assets/" + dest
		}

		if isLink {
			link.Destination = []byte(dest)
		} else {
			img.Destination = []byte(dest)
		}
		return ast.GoToNext
	})
	return document
}

func transpileMarkdown(source string) (string, error) {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	document := p.Parse([]byte(source))
	document = fixRelativeFilPaths(document)
	options := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(options)
	output := markdown.Render(document, renderer)
	return indentHtml(output, 8), nil

}

func buildPost(t *template.Template, inPath string, outPath string) error {
	source, err := readFile(inPath)
	if err != nil {
		return err
	}

	post, err := parsePostHeader(source)
	if err != nil {
		return err
	}

	content := source[post.headerIndex:]
	output, err := transpileMarkdown(content)
	if err != nil {
		return err
	}
	post.Content = template.HTML(output)

	post.StylesPath = "assets/styles.css"
	if outPath != "index.html" {
		post.StylesPath = "../" + post.StylesPath
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer file.Close()
	return t.Execute(file, post)
}

func newTemplate(path, id string) (*template.Template, error) {
	templateSource, err := readFile(path)
	if err != nil {
		return nil, err
	}
	t := template.New(id)
	t = template.Must(t.Parse(templateSource))
	return t, nil
}

func buildPosts(t *template.Template) error {
	entries, err := os.ReadDir("posts")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		pathParts := strings.Split(entry.Name(), ".")
		inPath := fmt.Sprintf("posts/%s", entry.Name())
		outPath := fmt.Sprintf("html/%s.html", pathParts[0])

		// Github pages requires the index.html to be in the branch root
		if pathParts[0] == "index" {
			outPath = fmt.Sprintf("%s.html", pathParts[0])
		}

		err = buildPost(t, inPath, outPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	t, err := newTemplate("assets/template.html", "Article")
	if err != nil {
		panic(err)
	}
	err = buildPosts(t)
	if err != nil {
		panic(err)
	}
}
