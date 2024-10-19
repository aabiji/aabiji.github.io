// A tiny static site generator -- it basically just
// transpiles all the markdown files in the posts folder
// to html and outputs them to the html folder. It's really basic,
// but it works well for my purposes. The workflow involves
// writing a markdown file in the posts folder, then running
// `go run .` to build the site.
// TODO; use filepath.Join
// TODO: maybe be smarter and automatically delete unused files
// TODO: better blog ui
// TODO: what if the assets were linking to don't exist?
// TODO: I also don't want to have to express certain parts of the ui in the template
// TODO: add code syntax highlighting

package main

import (
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
	file, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	contents := string(file)
	return strings.TrimSpace(contents), nil
}

func onlyContains(str string, char byte) bool {
	for i := 0; i < len(str); i++ {
		if str[i] != char {
			return false
		}
	}
	return true
}

func getFileParts(path string) (string, string) {
	pathParts := strings.Split(path, "/")
	file := pathParts[len(pathParts)-1]
	fileParts := strings.Split(file, ".")
	// Return the file name and extension
	return fileParts[0], fileParts[1]
}

func indentLines(content []byte, numSpaces int) string {
	indent := strings.Repeat(" ", numSpaces)
	lines := strings.Split(string(content), "\n")
	output := ""
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

func transpileMarkdown(source string, modifier func(ast.Node) ast.Node) (string, error) {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	document := p.Parse([]byte(source))
	document = modifier(document)
	options := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(options)
	output := markdown.Render(document, renderer)
	return indentLines(output, 8), nil

}

type Templates = map[string]*template.Template

func loadTemplates(folder string) (Templates, error) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	templates := make(Templates, 1)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := folder + "/" + entry.Name()
		source, err := readFile(path)
		if err != nil {
			return nil, err
		}

		t := template.New(entry.Name())
		t = template.Must(t.Parse(source))
		base, _ := getFileParts(entry.Name())
		templates[base] = t
	}

	return templates, nil
}

type Post struct {
	Content           template.HTML
	contentStartIndex int
	Info              map[string]string
	IsMainPage        bool
	StylesPath        string
}

// Parse the header at the top of each blog post.
// Each header is enclosed by horizantal rules.
// This would be an example of a header:
//
// ---
// Title: Something
// Date: Some date
// ---
func parsePostHeader(source string) (Post, error) {
	post := Post{Info: make(map[string]string, 1)}
	lines := strings.Split(source, "\n")
	inHeader := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 { // Ignore empty lines
			continue
		}

		post.contentStartIndex += len(line) + 1
		isHorizantalRule := onlyContains(line, '-')
		if isHorizantalRule && !inHeader {
			inHeader = true
			continue // Skip the opening ---
		} else if isHorizantalRule && inHeader {
			break // Stop on the ending ---
		}

		// Parse the key value pairs in the header
		if inHeader {
			pair := strings.Split(line, ":")
			if len(pair) != 2 {
				return Post{}, fmt.Errorf("invalid key value pair: %s", line)
			}
			key := strings.ToLower(strings.TrimSpace(pair[0]))
			value := strings.Trim(pair[1], " ")
			post.Info[key] = value

			if key == "title" && strings.ToLower(value) == "main" {
				post.IsMainPage = true
			}
		}
	}

	if post.contentStartIndex == len(source) {
		return Post{}, fmt.Errorf("post does not contain a header")
	}
	return post, nil
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
		base, extension := getFileParts(dest)
		if extension == "md" {
			dest = fmt.Sprintf("html/%s.html", base)
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

func buildPost(templates Templates, inPath string, outPath string) error {
	source, err := readFile(inPath)
	if err != nil {
		return err
	}

	post, err := parsePostHeader(source)
	if err != nil {
		return err
	}

	source = source[post.contentStartIndex:]
	output, err := transpileMarkdown(source, fixRelativeFilPaths)
	if err != nil {
		return err
	}
	post.Content = template.HTML(output)

	post.StylesPath = "assets/styles.css"
	if !post.IsMainPage {
		post.StylesPath = "../" + post.StylesPath
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer file.Close()

	id, exists := post.Info["template"]
	if !exists {
		return fmt.Errorf("%s : template not specified", inPath)
	}

	t, exists := templates[id]
	if !exists {
		return fmt.Errorf("%s is not a template", id)
	}

	return t.Execute(file, post)
}

// TODO: refactor this
func buildPosts() error {
	templates, err := loadTemplates("templates")
	if err != nil {
		return err
	}

	entries, err := os.ReadDir("posts")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		base, extension := getFileParts(entry.Name())
		if extension != "md" {
			continue
		}

		inPath := fmt.Sprintf("posts/%s", entry.Name())
		outPath := fmt.Sprintf("html/%s.html", base)
		if base == "index" {
			// Github pages requires the index.html to be in the branch root
			outPath = fmt.Sprintf("%s.html", base)
		}

		err = buildPost(templates, inPath, outPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	err := buildPosts()
	if err != nil {
		panic(err)
	}
}
