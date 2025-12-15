package utils

import (
	"bytes"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var (
	policy = bluemonday.UGCPolicy().
		AllowAttrs("class").OnElements("code", "pre", "span").
		AllowAttrs("target", "rel").OnElements("a").
		AddTargetBlankToFullyQualifiedLinks(true)

	md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(false),
				),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)
)

func RenderSafeHTML(src []byte) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return "", err
	}
	safe := policy.SanitizeBytes(buf.Bytes())
	return string(safe), nil
}
