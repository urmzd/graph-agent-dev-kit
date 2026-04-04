package eval

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompts/judge.prompt
var judgePromptRaw string

//go:embed prompts/judge_pairwise.prompt
var judgePairwisePromptRaw string

var (
	judgeTmpl         = template.Must(template.New("judge").Parse(judgePromptRaw))
	judgePairwiseTmpl = template.Must(template.New("judge_pairwise").Parse(judgePairwisePromptRaw))
)

func renderPrompt(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}
