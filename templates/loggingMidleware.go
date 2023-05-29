package templates

import (
	"text/template"
)

var LoggingTemplate *template.Template

func init() {
	LoggingTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(loggingTemplStr))
}

var loggingTemplStr = `
package middleware

import (
	"context"
	"fmt"
	"time"
	"{{ .PackagePath}}"
	"go.uber.org/zap"
)

// Middleware describes a service middleware.
type Middleware func(service {{ .ServicePackage }}.{{ .ServiceName }}) {{ .ServicePackage }}.{{ .ServiceName }}

func LoggingMiddleware(logger *zap.Logger) Middleware {
	return func(next {{ .ServicePackage }}.{{ .ServiceName }}) {{ .ServicePackage }}.{{ .ServiceName }} {
		return &loggingMiddleware{
			next:   next,
			logger: logger,
		}
	}
}

type loggingMiddleware struct {
	next   {{ .ServicePackage }}.{{ .ServiceName }}
	logger *zap.Logger
}

{{ range .Functions}}
// {{ .Name }} implements {{ $.ServicePackage }}.{{ $.ServiceName }}
func (mw *loggingMiddleware) {{ .Name }}{{ .Signature }}{
	defer func(begin time.Time) {
		mw.logger.Sugar().Info(
			"method: ",
			"{{ .Name }}",
			{{ range $index, $argument := .Arguments}}
			"{{first_letter_upper $argument.Name }}: ", fmt.Sprintf("%v ", {{ $argument.Name }}),
			{{end}}
			"time: ", fmt.Sprintf("%v ", time.Since(begin)),
		)
	}(time.Now())
	return mw.next.{{ .Name }}(ctx, {{ range $index, $argument := .Arguments}}{{ $argument.Name }},{{end}})
}
{{ end }}
`
