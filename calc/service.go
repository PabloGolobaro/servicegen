package services

import (
	"context"
)

//go:generate servicegen

//servicegen:service http nats
type CalcService interface {
	Add(ctx context.Context, a, b int) (int, error)
	Erase(ctx context.Context, User string, Mail string) (uint, error)
}
