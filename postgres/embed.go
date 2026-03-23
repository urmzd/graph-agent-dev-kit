package postgres

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed sql/migrations.sql.tmpl
var migrationsRaw string

var migrationsTmpl = template.Must(template.New("migrations").Parse(migrationsRaw))

func renderTemplate(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic("render template: " + err.Error())
	}
	return buf.String()
}
