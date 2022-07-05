package utils

import (
	"bytes"
	"encoding/base64"
	"text/template"
)

func render(t *template.Template, data interface{}) string {
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func RenderString(tmp string, data interface{}) string {
	return render(template.Must(template.New("_").Funcs(funcMap).Parse(tmp)), data)
}

var funcMap = template.FuncMap{
	"base64": base64.StdEncoding.EncodeToString,
}
