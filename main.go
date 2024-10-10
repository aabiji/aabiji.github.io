package main

import (
	"bufio"
	"errors"
	"html/template"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
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
	Content     string
	Info        map[string]string
	headerIndex int // Index where the post header ends
}

func parsePostHeader(source string) (Post, error) {
	post := Post{Info: make(map[string]string, 1)}
	lines := strings.Split(source, "\n")

	for _, line := range lines {
		// Ignore empty lines
		if len(strings.Trim(line, " ")) == 0 {
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

func transpileMarkdown(source string) (string, error) {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	document := p.Parse([]byte(source))
	options := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(options)
	output := markdown.Render(document, renderer)
	return string(output), nil
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
	post.Content, err = transpileMarkdown(content)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(outPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	t.Execute(writer, post)
	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	templateSource, err := readFile("template.html")
	if err != nil {
		panic(err)
	}

	t := template.New("Article")
	t = template.Must(t.Parse(templateSource))
	err = buildPost(t, "home.md", "home.html")
	if err != nil {
		panic(err)
	}
}
