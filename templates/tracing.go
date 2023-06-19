package templates

import (
	"strings"
	"text/template"
)

var TracingTemplate *template.Template

func init() {
	TracingStr = strings.Replace(TracingStr, "^", "`", -1)
	TracingTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(TracingStr))
}

var TracingStr = `
package otelTracing

import (
	"context"
	"net/http"	
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func newExporter() (trace.SpanExporter, error) {
	exporter, err := jaeger.New(
		jaeger.WithAgentEndpoint(jaeger.WithAgentHost("localhost")))
	if err != nil {
		return nil, err
	}
	return exporter, nil
}
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("AddServiceTest"),
			semconv.ServiceVersionKey.String("v0.1.0"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}

func InitTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := newExporter()
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, err
}


// ExtractTraceFromHttpHeaders is a function to use in ServerBefore middleware to get
// current span information from http Headers
func ExtractTraceFromHttpHeaders(ctx context.Context, request *http.Request) context.Context {
	h := request.Header
	extractCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(h))

	return extractCtx

}

// InjectTraceToHttpHeaders is a function to use in ClientBefore middleware to inject
// current span information to http Headers of invoking request
func InjectTraceToHttpHeaders(ctx context.Context, request *http.Request) context.Context {
	h := request.Header
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(h))

	return ctx
}

// ExtractTraceFromNatsHeaders is a function to use in ServerBefore middleware to get
// current span information from NATS Headers
func ExtractTraceFromNatsHeaders(ctx context.Context, msg *nats.Msg) context.Context {
	h := msg.Header
	extractCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(h))

	return extractCtx

}

// InjectTraceToNatsHeaders is a function to use in ClientBefore middleware to inject
// current span information to NATS Headers of invoking msg
func InjectTraceToNatsHeaders(ctx context.Context, msg *nats.Msg) context.Context {
	h := msg.Header
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(h))

	return ctx
}

`
