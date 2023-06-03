package templates

import "text/template"

var InstrumentationTemplate *template.Template

func init() {
	InstrumentationTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(instrumentationTemplStr))
}

var instrumentationTemplStr = `
package middleware

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"{{ .PackagePath}}"
	"time"
)

func InitInstrumentingMiddleware(svc {{ .ServicePackage }}.{{ .ServiceName }}) {{ .ServicePackage }}.{{ .ServiceName }} {

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "my_group",
		Subsystem: "{{ .ServicePackage }}.{{ .ServiceName }}",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "my_group",
		Subsystem: "{{ .ServicePackage }}.{{ .ServiceName }}",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	return instrumentingMiddleware{
		requestCount:   requestCount,
		requestLatency: requestLatency,
		next:           svc,
	}
}

type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           {{ .ServicePackage }}.{{ .ServiceName }}
}

{{ range .Functions}}

func (mw instrumentingMiddleware) {{ .Name }}{{ .Signature }} {
	{{ range $index, $result := .Results}}
	{{ if eq $index 0}}
	var output {{ $result.Type }}
	{{else}}
	var err {{ $result.Type }}
	{{end}}
	{{end}}
	defer func(begin time.Time) {
		lvs := []string{"method", "{{ lower .Name }}", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	output, err = mw.next.{{ .Name }}({{ range $index, $argument := .Arguments}}{{ $argument.Name }},{{end}})
	return output,err
}

{{end}}
`
