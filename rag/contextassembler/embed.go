package contextassembler

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompts/compression.prompt
var compressionRaw string

var compressionTmpl = template.Must(template.New("compression").Parse(compressionRaw))

func renderPrompt(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic("render prompt: " + err.Error())
	}
	return buf.String()
}
