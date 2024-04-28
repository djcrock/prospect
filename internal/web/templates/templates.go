package templates

import (
	"embed"
	"html/template"
	"io"
)

//go:embed *.tmpl
var files embed.FS

// Template wraps template.Template to provide a simpler interface.
type Template struct {
	*template.Template
}

func (t Template) ExecuteFull(w io.Writer, data any) error {
	return t.ExecuteTemplate(w, "base", data)
}

func (t Template) ExecutePartial(w io.Writer, data any) error {
	return t.ExecuteTemplate(w, "content", data)
}

var (
	Game     Template
	Index    Template
	NotFound Template
)

func init() {
	Game = getTemplate("game.tmpl")
	Index = getTemplate("index.tmpl")
	NotFound = getTemplate("not_found.tmpl")
}

func getTemplate(name string) Template {
	return Template{template.Must(template.ParseFS(files, name, "base.tmpl"))}
}
