package templates

import "text/template"

var ServiceTemplate = template.Must(template.New("").Parse(`
package implementation

import (
	"context"
	"log"
	
)

// {{ .ServiceName }}Service implements the {{ .ServicePackage }}.{{ .ServiceName }}
type {{ .ServiceName }}Service struct {
	logger *log.Logger
}

func New{{ .ServiceName }}Service(logger *log.Logger) {{ .ServicePackage }}.{{ .ServiceName }} {
	return &{{ .ServiceName }}Service{
		logger: logger,
	}
}

{{ range .Functions}}
// {{ .Name }} implements {{ $.ServicePackage }}.{{ $.ServiceName }}
func (s *{{ $.ServiceName }}Service){{ .Name }} {{ .Signature }} {

	panic("Not implemented yet")
}

{{end}}

`))
