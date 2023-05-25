package templates

import (
	"strings"
	"text/template"
)

var TransportTemplate *template.Template
var LowerCaseFunc = func(str string) string {
	return strings.ToLower(str)
}
var UpperFirstLetter = func(str string) string {
	return strings.Title(str)
}
var templStr = `
package transport

import (
	"context"
	"github.com/go-kit/kit/endpoint"
)

// Endpoints holds all Go kit endpoints for the {{ .ServicePackage }}.{{ .ServiceName }}
type Endpoints struct {
	{{ range .Functions}}
	{{ .Name }} endpoint.Endpoint
	{{end}}
}

// MakeEndpoints initializes all Go kit endpoints for the {{ .ServicePackage }}.{{ .ServiceName }}.
func MakeEndpoints(s {{ .ServicePackage }}.{{ .ServiceName }}) Endpoints {
	return Endpoints{
		{{ range .Functions}}
		{{ .Name }}: make{{ .Name }}Endpoint(s),
		{{end}}
	}
}

{{ range .Functions}}
func make{{ .Name }}Endpoint(s {{ $.ServicePackage }}.{{ $.ServiceName }}) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.({{ .Name }}Request) // type assertion
		res, err := s.{{ .Name }}(ctx, {{ range $index, $argument := .Arguments}}req.{{first_letter_upper $argument.Name }},{{end}})
		if err != nil {
			return {{ .Name}}Response{Success: false, Error: services.NewAppError(err)}, nil
		}
		return {{ .Name}}Response{Success: true, Result: res, Error: nil}, nil
	}
}

{{ end }}

// GenericErrorResponse holds the success result and error
type GenericErrorResponse struct {
	Success bool               ^json:"success"^
	Error   *services.AppError ^json:"error,omitempty"^
}

{{ range .Functions}}
// {{ .Name }}Request holds the request parameters for the {{ .Name }} method.
type {{ .Name }}Request struct {
	{{ range $index, $argument := .Arguments}}
	{{first_letter_upper $argument.Name }} {{ $argument.Type }} ^json:"{{ lower $argument.Name }}"^
{{end}}
}

// {{ .Name }}Response holds the response values for the {{ .Name }} method.
type {{ .Name }}Response struct {
	Success bool                ^json:"success"^
	Result {{ .ResultType }}     ^json:"result"^
	Error *services.AppError ^json:"error,omitempty"^
}

func (r {{ .Name }}Response) Failed() error {
	if r.Error == nil {
		return nil
	}
	return r.Error.E
}

func (r {{ .Name }}Response) IsRetryable() bool {
	return r.Error.IsRetryable()
}
{{end}}
`

func init() {
	templStr = strings.Replace(templStr, "^", "`", -1)
	TransportTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(templStr))

}
