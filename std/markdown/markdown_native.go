package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"

	. "github.com/candid82/joker/core"
)

func convertString(source string) string {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.DefinitionList,
			extension.Footnote,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // allow for raw markup
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}
	return buf.String()
}

func getKeywordFlag(opts Map, name string, def bool) bool {
	ok, entry := opts.Get(MakeKeyword(name))
	if !ok {
		return def
	}
	val, ok := entry.(Boolean)
	if !ok {
		panic(RT.NewError("flags must be a boolean"))
	}
	return val.B
}

func convertStringOpts(source string, opts Object) string {
	options, ok := opts.(Map)
	if !ok {
		panic(RT.NewError("Options must be a map"))
	}

	renderOptions := []renderer.Option{}
	if flag := getKeywordFlag(options, "with-hard-wraps?", true); flag {
		renderOptions = append(renderOptions, html.WithHardWraps())
	}
	if flag := getKeywordFlag(options, "with-xhtml?", true); flag {
		renderOptions = append(renderOptions, html.WithXHTML())
	}
	if flag := getKeywordFlag(options, "with-unsafe?", true); flag {
		renderOptions = append(renderOptions, html.WithUnsafe())
	}
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.DefinitionList,
			extension.Footnote,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(renderOptions...),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		panic(err)
	}
	return buf.String()
}
