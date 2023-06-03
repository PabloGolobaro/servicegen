package calc

import (
	"context"
)

//go:generate servicegen -mod github.com/pablogolobaro/servicegen

//servicegen:service http nats logging tracing
type Calc interface {
	Add(ctx context.Context, a, b int) (int, error)
	Erase(ctx context.Context, User string, Mail string) (uint, error)
}
