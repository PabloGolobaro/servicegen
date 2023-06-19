package generator

import (
	"bytes"
	"fmt"
	"github.com/pablogolobaro/servicegen/templates"
)

const (
	ImplementationPackage = "implementation"
	CmdPackage            = "cmd"
	OtelTracingPackage    = "otelTracing"
	ConfigPackage         = "config"
	RootFilename          = "root"
	HttpRunFilename       = "httprun"
	TransportPackage      = "transport"
	HttpPackage           = "httptransport"
	NatsPackage           = "natstransport"
	MiddlewarePackage     = "middleware"
	HttpFileName          = "http"
	NatsFileName          = "nats"
	LoggingFileName       = "logging"
	TracingFileName       = "tracing"
	ErrorFileName         = "error"
)

type templateParams struct {
	ServiceName      string
	ServicePackage   string
	Functions        []ServiceFunction
	PackagePath      string
	TransportPackage string
	ModuleName       string
}

func (r ServiceGenerator) ExecuteTemplate(buf *bytes.Buffer, packageName string, fileName string, params templateParams) error {
	var err error

	switch packageName {
	case ImplementationPackage:
		err = templates.ServiceTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case TransportPackage:
		err = templates.TransportTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case HttpPackage:
		err = templates.HttpTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case ConfigPackage:
		err = templates.ConfigTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case OtelTracingPackage:
		err = templates.TracingTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case NatsPackage:
		err = templates.NatsTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case r.ServicePackageName:
		err = templates.ErrorTemplate.Execute(buf, params)
		if err != nil {
			return fmt.Errorf("execute template: %v", err)
		}
	case MiddlewarePackage:

		switch fileName {
		case LoggingFileName:
			err = templates.LoggingTemplate.Execute(buf, params)
			if err != nil {
				return fmt.Errorf("execute template: %v", err)
			}
		case TracingFileName:
			err = templates.InstrumentationTemplate.Execute(buf, params)
			if err != nil {
				return fmt.Errorf("execute template: %v", err)
			}

		default:
			return fmt.Errorf("execute template: unknown fileName")
		}
	case CmdPackage:
		switch fileName {
		case RootFilename:
			err = templates.RootTemplate.Execute(buf, params)
			if err != nil {
				return fmt.Errorf("execute template: %v", err)
			}
		case HttpRunFilename:
			err = templates.HttpRunTemplate.Execute(buf, params)
			if err != nil {
				return fmt.Errorf("execute template: %v", err)
			}

		default:
			return fmt.Errorf("execute template: unknown fileName")
		}

	default:
		return fmt.Errorf("execute template: unknown packageName")
	}

	return err
}
