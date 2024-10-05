package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Token types
const (
	HASH           = 0
	ASTRIX         = 1
	TILDE          = 2
	DASH           = 3
	BACKTICK       = 4
	BACKSLASH      = 5
	CARET          = 6
	PIPE           = 7
	EXCALAMTION    = 8
	COLON          = 9
	OPEN_PAREN     = 10
	CLOSED_PAREN   = 11
	OPEN_BRACKET   = 12
	CLOSED_BRACKET = 13
	ANGLE_BRACKET  = 14
	SPACE          = 15
	NEWLINE        = 16
	ORDERING       = 17
	WORD           = 18
	EOF            = 19
)

type token struct {
	raw string
	id  int
}

// Check if a token is one of many types
func (t *token) is(possibleTypes ...int) bool {
	for _, i := range possibleTypes {
		if t.id == i {
			return true
		}
	}
	return false
}

func isOrdering(str string) bool {
	// An ordering is anything that looks like this:
	// 1.
	// Any number immediately followed by a dot.

	prefix := str[:len(str)-1] // Remove the potential dot
	_, err := strconv.Atoi(prefix)
	return err == nil && str[len(str)-1] == '.'
}

func tokenize(characters []rune, tokenTypes map[rune]int) []token {
	start := 0
	var tokens []token

	for i, char := range characters {
		// If we have a token, output it directly
		value, currentIsToken := tokenTypes[char]
		if currentIsToken {
			tokens = append(tokens, token{raw: string(char), id: value})
			start = i + 1
			continue
		}

		nextIsToken := false
		if i == len(characters)-1 {
			nextIsToken = true // Reached the EOF token
		} else {
			_, nextIsToken = tokenTypes[characters[i+1]]
		}

		// We'll group the characters in between tokens together as a word
		if nextIsToken {
			text := string(characters[start : i+1])
			tokenType := WORD
			if isOrdering(text) {
				tokenType = ORDERING
			}
			tokens = append(tokens, token{raw: text, id: tokenType})
			start = i + 1
		}
	}

	return tokens
}

// What's a easier, cleaner way to parse???
// Our parsing (especially text) feels like a bunch of hacks
// This is harder than I thought it would be. We can either continue to parse,
// or we can just use gomarkdown and inject our own custom parser
// TODO: try to break our parser
// Markdown parser. We'll need
// - Lexer
// - Parser that parses into html nodes direclty
// - Output generation (that we can apply a transformation to as were walking the tree)
//   convert html nodes into a string (with indentation)
// TODO; create html node
// Most things should eventualy boil down to <p> tags
// However should we remove the excess tags during output generation?

/*
Node:
type : element, text
content : raw text
dataAtom :
*/

func newNode(nodeAtom atom.Atom) *html.Node {
	node := new(html.Node)
	if nodeAtom == atom.P {
		node.Type = html.TextNode
	} else {
		node.Type = html.ElementNode
	}
	node.DataAtom = nodeAtom
	return node
}

type tokenParser struct {
	tokens []token
	index  int
}

func (p *tokenParser) nextToken(offset int) token {
	if p.index+offset < len(p.tokens) {
		return p.tokens[p.index+offset]
	}
	return token{id: EOF}
}

func (p *tokenParser) prevToken() token {
	if p.index-1 < 0 {
		return token{id: EOF}
	}
	return p.tokens[p.index-1]
}

func (p *tokenParser) parseEscaped() string {
	p.index += 1 // Skip the backslash
	text := ""

	// Keep reading tokens until we hit a word, space
	// newline or end of file
	for {
		current := p.nextToken(0)
		if current.is(WORD, SPACE, NEWLINE, EOF) {
			break
		}
		text += current.raw
		p.index += 1
	}

	return text
}

func (p *tokenParser) currentIsTextual() bool {
	current := p.nextToken(0)
	return !current.is(EOF, BACKSLASH, NEWLINE)
}

// TODO: rewrite this
func (p *tokenParser) parseText() *html.Node {
	node := newNode(atom.P)

	for {
		current := p.nextToken(0)
		next := p.nextToken(1)

		// Paragraphs are separated by empty lines
		if current.id == NEWLINE && next.id == NEWLINE {
			break
		}

		// Ignore whitespace
		if current.is(SPACE, NEWLINE) {
			if len(node.Data) > 0 && node.Data[len(node.Data)-1] != ' ' {
				node.Data += " "
			}
			p.index += 1
			continue
		}

		// Read text
		if current.id == WORD || p.currentIsTextual() {
			node.Data += current.raw
			p.index += 1
			continue
		} else if current.id == BACKSLASH {
			node.Data += p.parseEscaped()
			continue
		}

		break // Stop when we hit an unwanted token
	}

	return node
}

func (p *tokenParser) parseHeader() *html.Node {
	count := 0
	for p.nextToken(0).id == HASH {
		count += 1
		p.index += 1
	}

	// If there's no space following the #, then it's just text
	if p.nextToken(0).id != SPACE {
		text := strings.Repeat("#", count)
		node := p.parseText()
		node.Data = text + node.Data
		return node
	}

	// Create a header node
	headers := []atom.Atom{atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6}
	node := newNode(headers[count-1])
	p.appendChildren(node, NEWLINE)
	return node
}

func (p *tokenParser) parseUntil(delimeter int) *html.Node {
	node := newNode(atom.Div)

	for p.index < len(p.tokens) && p.tokens[p.index].id != delimeter {
		current := p.nextToken(0)
		previous := p.prevToken()

		onNewline := previous.is(EOF, NEWLINE)
		if current.id == HASH && onNewline {
			node.AppendChild(p.parseHeader())
		} else if !current.is(EOF, NEWLINE) { // TODO: dodgy!
			node.AppendChild(p.parseText())
		}

		p.index += 1
	}

	return node
}

func (p *tokenParser) appendChildren(node *html.Node, delimeter int) {
	// Recursively parse, then move child elements out of the container div into the node
	div := p.parseUntil(delimeter)
	for child := div.FirstChild; child != nil; child = child.NextSibling {
		div.RemoveChild(child)
		node.AppendChild(child)
	}
}

func printAST(node *html.Node, depth int) {
	indent := strings.Repeat(" ", depth*2)
	fmt.Println(indent, node.DataAtom.String(), node.Data)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		printAST(child, depth+1)
	}
}

func main() {
	file, err := os.ReadFile("test.md")
	if err != nil {
		panic(err)
	}
	characters := []rune(string(file))

	parser := tokenParser{index: 0}
	tokenTypes := map[rune]int{
		'#': HASH, '*': ASTRIX, '~': TILDE, '-': DASH, '`': BACKTICK,
		'^': CARET, '|': PIPE, '!': EXCALAMTION, '(': OPEN_PAREN, ')': CLOSED_PAREN,
		'[': OPEN_BRACKET, ']': CLOSED_BRACKET, '>': ANGLE_BRACKET, ':': COLON,
		' ': SPACE, '\n': NEWLINE, '\\': BACKSLASH,
	}
	parser.tokens = tokenize(characters, tokenTypes)
	printAST(parser.parseUntil(EOF), 0)
}
