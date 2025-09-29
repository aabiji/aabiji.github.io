// A simple static site generator. It transpiles all the markdown
// files in the posts/ folder to html files then outputs them into the
// html/ folder.

package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
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

func ensureFolderExists(path string) error {
	// Create the output directory if it doesn't already exist
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func getFileParts(path string) (string, string) {
	pathParts := strings.Split(path, "/")
	file := pathParts[len(pathParts)-1]
	fileParts := strings.Split(file, ".")
	// Return the file name and extension
	return fileParts[0], fileParts[1]
}

type Transformer func(ast.Node) (ast.Node, error)

func markdownToHTML(source string, t Transformer) (string, error) {
	p := parser.NewWithExtensions(parser.CommonExtensions)
	document := p.Parse([]byte(source))
	document, err := t(document)
	if err != nil {
		return "", err
	}
	options := html.RendererOptions{Flags: html.CommonFlags}
	renderer := html.NewRenderer(options)
	output := markdown.Render(document, renderer)
	return string(output), nil
}

var (
	ASSET_FOLDER       = "assets"
	POSTS_FOLDER       = "posts"
	TEMPLATE_PATH      = "assets/template.html"
	OUTPUT_FOLDER      = "html"
	TEMP_OUTPUT_FOLDER = ""
)

func getDestination(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Link:
		return string(n.Destination)
	case *ast.Image:
		return string(n.Destination)
	default:
		return ""
	}
}

func setDestination(node ast.Node, dest string) {
	switch n := node.(type) {
	case *ast.Link:
		n.Destination = []byte(dest)
	case *ast.Image:
		n.Destination = []byte(dest)
	}
}

func fixRelativeFilPaths(document ast.Node) (ast.Node, error) {
	var problem error
	ast.WalkFunc(document, func(node ast.Node, entering bool) ast.WalkStatus {
		path := getDestination(node)
		if path == "" || !entering {
			return ast.GoToNext
		}

		if _, err := url.ParseRequestURI(path); err == nil {
			return ast.GoToNext // Ignore valid urls
		}

		realPath := ""
		base, extension := getFileParts(path)
		if extension == "md" {
			realPath = fmt.Sprintf("%s/%s.html", OUTPUT_FOLDER, base)
		} else {
			realPath = fmt.Sprintf("%s/%s", ASSET_FOLDER, path)
		}
		setDestination(node, "/"+realPath)

		// We should check if the markdown file is present and not its html output
		if extension == "md" {
			realPath = fmt.Sprintf("%s/%s.md", POSTS_FOLDER, base)
		}

		if _, err := os.Stat(realPath); errors.Is(err, os.ErrNotExist) {
			problem = err
			return ast.Terminate
		}
		return ast.GoToNext
	})
	return document, problem
}

type Post struct {
	HTMLContent template.HTML
	StylePath   string
	ArticlePage bool
	Title       string
	inPath      string
	outPath     string
}

func newPost(markdownFile string) (Post, error) {
	source, err := readFile(markdownFile)
	if err != nil {
		return Post{}, err
	}
	lineEnd := strings.Index(source, "\n")
	firstLine := source[:lineEnd]

	base, _ := getFileParts(markdownFile)
	post := Post{
		inPath:      markdownFile,
		outPath:     fmt.Sprintf("%s/%s.html", TEMP_OUTPUT_FOLDER, base),
		Title:       firstLine[2:],
		StylePath:   fmt.Sprintf("%s/styles.css", ASSET_FOLDER),
		ArticlePage: base != "index",
	}

	if base != "index" {
		post.StylePath = "../" + post.StylePath
	}

	output, err := markdownToHTML(source, fixRelativeFilPaths)
	if err != nil {
		return post, err
	}

	post.HTMLContent = template.HTML(output)
	return post, err
}

func buildPost(post *Post, t *template.Template) error {
	err := ensureFolderExists(post.outPath)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(post.outPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}

	defer file.Close()
	return t.Execute(file, post)
}

func buildPosts() error {
	templateSource, err := readFile(TEMPLATE_PATH)
	if err != nil {
		return err
	}
	t := template.New("Site template")
	t = template.Must(t.Parse(templateSource))

	entries, err := os.ReadDir(POSTS_FOLDER)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, extension := getFileParts(entry.Name()); extension != "md" {
			continue
		}

		inPath := fmt.Sprintf("%s/%s", POSTS_FOLDER, entry.Name())
		post, err := newPost(inPath)
		if err != nil {
			return err
		}

		err = buildPost(&post, t)
		if err != nil {
			return err
		}
	}

	return nil
}

// Move files from the temporary folder to the actual output folder
func moveFiles() error {
	entries, err := os.ReadDir(TEMP_OUTPUT_FOLDER)
	if err != nil {
		return err
	}

	os.RemoveAll(OUTPUT_FOLDER)
	os.Mkdir(OUTPUT_FOLDER, os.ModePerm)

	for _, entry := range entries {
		inPath := fmt.Sprintf("%s/%s", TEMP_OUTPUT_FOLDER, entry.Name())
		outPath := fmt.Sprintf("%s/%s", OUTPUT_FOLDER, entry.Name())
		// Github pages requires the index.html file to be in the repository root
		if entry.Name() == "index.html" {
			outPath = "index.html"
		}

		contents, err := os.ReadFile(inPath)
		if err != nil {
			return err
		}
		err = os.WriteFile(outPath, contents, 0664)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	var err error

	// We write the built html files to a temporary folder so that no
	// existing html files are corrupted if something goes wrong
	TEMP_OUTPUT_FOLDER, err = os.MkdirTemp("", "html")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = buildPosts()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = moveFiles()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
