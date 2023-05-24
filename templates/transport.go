package templates

import "text/template"

var TransportTemplate = template.Must(template.New("").Parse(`
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
		res, err := s.{{ .Name }}(ctx, {{ range $index, $argument := .Arguments}}req.{{ $argument.Name }},{{end}})
		if err != nil {
			return AddResponse{Success: false, Error: services.NewAppError(err)}, nil
		}
		return AddResponse{Success: true, Result: res, Error: nil}, nil
	}
}

{{ end }}

// GenericErrorResponse holds the success result and error
type GenericErrorResponse struct {
	Success bool               
	Error   *services.AppError 
}

{{ range .Functions}}
// {{ .Name }}Request holds the request parameters for the {{ .Name }} method.
type {{ .Name }}Request struct {
	{{ range $index, $argument := .Arguments}}
	{{ $argument.Name }} {{ $argument.Type }}
{{end}}
}

// {{ .Name }}Response holds the response values for the {{ .Name }} method.
type {{ .Name }}Response struct {
	Success bool               
	Result  int                
	Error   *services.AppError 
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
`))
