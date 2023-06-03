package generator

import (
	"bytes"
	"fmt"
	"github.com/pablogolobaro/servicegen/templates"
)

const (
	ImplementationPackage = "implementation"
	TransportPackage      = "transport"
	HttpPackage           = "httptransport"
	NatsPackage           = "natstransport"
	MiddlewarePackage     = "middleware"
	HttpFileName          = "http"
	NatsFileName          = "nats"
	LoggingFileName       = "logging"
	TracingFileName       = "tracing"
)

type templateParams struct {
	ServiceName      string
	ServicePackage   string
	Functions        []ServiceFunction
	TransportPackage string
	PackagePath      string
}

func ExecuteTemplate(buf *bytes.Buffer, packageName string, fileName string, params templateParams) error {
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
	case NatsPackage:
		err = templates.NatsTemplate.Execute(buf, params)
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
	}

	return err
}
