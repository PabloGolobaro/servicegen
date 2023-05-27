package templates

import (
	"text/template"
)

var NatsTemplate *template.Template

func init() {
	NatsTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(natsTemplStr))
}

var natsTemplStr = `
package trnats

import (
	kitzap "github.com/go-kit/kit/log/zap"
	kittransport "github.com/go-kit/kit/transport"
	"gitlab.pluspay.ru/chestnut/servicepot/pkg/otelTracing"
	"gitlab.pluspay.ru/chestnut/servicepot/services/calc/transport"
	"go.uber.org/zap/zapcore"

	kitnats "github.com/go-kit/kit/transport/nats"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const ADD_SUBJECT = "add"

func RegisterSubscribers(svcEndpoints transport.Endpoints, logger *zap.Logger, conn *nats.Conn) error {
	options := []kitnats.SubscriberOption{
		kitnats.SubscriberErrorHandler(kittransport.NewLogErrorHandler(kitzap.NewZapSugarLogger(logger, zapcore.ErrorLevel))),
		kitnats.SubscriberErrorEncoder(encodeErrorResponse),
		kitnats.SubscriberBefore(otelTracing.ExtractTraceFromNatsHeaders),
	}
	{{ range .Functions}}
	{{lower .Name}} := kitnats.NewSubscriber(
		svcEndpoints.{{ .Name}},
		decode{{ .Name}}Request,
		encode{{.Name}}Response,
		options...,
	).ServeMsg(conn)
	{{end}}

{{ range .Functions}}
	_, err := conn.QueueSubscribe({{lower .Name}}, "", {{lower .Name}})
{{end}}
	return err
}

{{ range .Functions}}
func decode{{.Name}}Request(ctx context.Context, msg *nats.Msg) (request interface{}, err error) {
	var req transport.{{.Name}}Request

	if e := json.Unmarshal(msg.Data, &req); e != nil {
		return nil, e
	}
	return req, nil
}

func encode{{.Name}}Response(ctx context.Context, q string, nc *nats.Conn, response interface{}) error {

	if e, ok := response.(endpoint.Failer); ok && e.Failed() != nil {
		// Not a Go kit transport error, but a business-logic error.
		// Provide those as HTTP errors.
		encodeErrorResponse(ctx, e.Failed(), q, nc)
		return nil
	}
	res, _ := json.Marshal(response)
	return nc.Publish(q, res)
}

{{end}}
func encodeErrorResponse(ctx context.Context, err error, q string, nc *nats.Conn) {
	resp, _ := json.Marshal(transport.GenericErrorResponse{Success: false, Error: services.NewAppError(err)})
	nc.Publish(q, resp)
}


`
