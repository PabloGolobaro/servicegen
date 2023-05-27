package templates

import (
	"text/template"
)

var HttpTemplate *template.Template

func init() {
	HttpTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(httpTemplStr))
}

var httpTemplStr = `
package http

import (
	kitzap "github.com/go-kit/kit/log/zap"
	kittransport "github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/labstack/echo/v4"
	"gitlab.pluspay.ru/chestnut/servicepot/pkg/otelTracing"
	"gitlab.pluspay.ru/chestnut/servicepot/services/calc/transport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func RegisterEndpoints(svcEndpoints transport.Endpoints, logger *zap.Logger, g *echo.Group) error {
	options := []kithttp.ServerOption{}
	// set-up router and initialize http endpoints
	var (
		errorLogger  = kithttp.ServerErrorHandler(kittransport.NewLogErrorHandler(kitzap.NewZapSugarLogger(logger, zapcore.ErrorLevel)))
		errorEncoder = kithttp.ServerErrorEncoder(encodeErrorResponse)
		before       = kithttp.ServerBefore(otelTracing.ExtractTraceFromHttpHeaders)
	)
	options = append(options, errorLogger, errorEncoder, before)


{{ range .Functions}}

	g.GET("/{{ lower .Name }}", echo.WrapHandler(kithttp.NewServer(
		svcEndpoints.{{ .Name}},
		decode{{ .Name}}Request,
		encode{{ .Name}}Response,
		options...,
	)))
{{ end }}

	return nil
}

{{ range .Functions}}
func decode{{ .Name}}Request(_ context.Context, r *http.Request) (request interface{}, err error) {
	var req transport.{{ .Name}}Request
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		return nil, e
	}
	return req, nil
}

func encode{{ .Name}}Response(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(endpoint.Failer); ok && e.Failed() != nil {
		// Not a Go kit transport error, but a business-logic error.
		// Provide those as HTTP errors.
		encodeErrorResponse(ctx, e.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

{{ end }}
func encodeErrorResponse(_ context.Context, err error, w http.ResponseWriter) {
	if err == nil {
		panic("encodeError with nil error")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	//w.WriteHeader()
	json.NewEncoder(w).Encode(transport.GenericErrorResponse{Success: false, Error: services.NewAppError(err)})
}

`
