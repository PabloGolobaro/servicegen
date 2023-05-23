package implementation

import (
	"context"
	services "github.com/pablogolobaro/servicegen/calc"
	"log"
)

// CalcServiceService implements the services.CalcService
type CalcServiceService struct{ logger *log.Logger }

func NewCalcServiceService(logger *log.Logger) services.CalcService {
	return &CalcServiceService{logger: logger}
}

func (s *CalcServiceService) Add(ctx context.Context, a, b int) (int, error) {
	panic(any("Not implemented yet"))
}

// Erase implements services.CalcService
func (s *CalcServiceService) Erase(ctx context.Context, User string, Mail string) (uint, error) {
	panic(any("Not implemented yet"))
}
