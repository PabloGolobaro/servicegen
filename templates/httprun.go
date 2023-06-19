package templates

import (
	"strings"
	"text/template"
)

var HttpRunTemplate *template.Template

func init() {
	HttpRunStr = strings.Replace(HttpRunStr, "^", "`", -1)
	HttpRunTemplate = template.Must(template.New("").Funcs(template.FuncMap{
		"lower":              LowerCaseFunc,
		"first_letter_upper": UpperFirstLetter,
	}).Parse(HttpRunStr))
}

var HttpRunStr = `/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"flag"
	"fmt"
	"{{ .PackagePath}}/otelTracing"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo-contrib/prometheus"
	"{{ .PackagePath}}"
	"{{ .PackagePath}}/implementation"
	"{{ .PackagePath}}/middleware"
	"{{ .PackagePath}}/transport"
	"{{ .PackagePath}}/transport/httptransport"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// httprunCmd represents the httprun command
var httprunCmd = &cobra.Command{
	Use:   "httprun",
	Short: "A brief description of your command",
	Long: "A longer description.",
	Run: func(cmd *cobra.Command, args []string) {
		Run(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(httprunCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// httprunCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// httprunCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func Run(cmd *cobra.Command, args []string) {
	var (
		httpAddr = flag.String("http.addr", ":8080", "HTTP listen address")
	)
	logger, _ := zap.NewDevelopmentConfig().Build()

	if tracingFlag {
		tp, err := otelTracing.InitTracer()
		if err != nil {
			logger.Sugar().Fatal(err)
		}
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				logger.Sugar().Infof("Error shutting down tracer provider: %v", err)
			}
		}()
	}

	var svc {{ .ServicePackage }}.{{ .ServiceName }}
	{

		svc = implementation.NewService(logger)

		svc = middleware.LoggingMiddleware(logger)(svc)

		svc = middleware.InitInstrumentingMiddleware(svc)
	}
	// Create Go kit endpoints for the Order Service
	// Then decorates with endpoint middlewares
	var endpoints transport.Endpoints
	{
		endpoints = transport.MakeEndpoints(svc)
		// add tracing middleware to endpoint
		{{ range .Functions}}
		endpoints.{{ .Name}} = otelkit.EndpointMiddleware(otelkit.WithOperation("{{ .Name}}Service"))(endpoints.{{ .Name}})
		{{end}}

	}

	server := echo.New()
	server.HideBanner = true

	p := prometheus.NewPrometheus("echo", nil)
	p.Use(server)

	g := server.Group("/api/v1")
	{

		err := httptransport.RegisterEndpoints(endpoints, logger, g)
		if err != nil {
			logger.Sugar().Info("Cannot Register Endpoints:", err)
			return
		}

	}

	errs := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		logger.Sugar().Info("transport", "HTTP", "addr", *httpAddr)

		errs <- server.Start(*httpAddr)
	}()

	logger.Sugar().Info("exit", <-errs)
}
`
