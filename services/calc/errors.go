package calc

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNegativeAddArgs = errors.New("add arguments cannot be negative")
	ErrTestRetryable   = errors.New("error to test retry")
)

var errCodes = map[error]int{
	ErrNegativeAddArgs: 333,
	ErrTestRetryable:   444,
}

var retryableErr = map[error]bool{
	ErrNegativeAddArgs: false,
	ErrTestRetryable:   true,
}

type AppError struct {
	E    error
	Code int `json:"code"`
}

func NewAppError(e error) *AppError {
	code, ok := errCodes[e]
	if !ok {
		code = 500
	}
	return &AppError{
		E:    e,
		Code: code,
	}
}

func (e AppError) Error() string {
	if e.E == nil {
		return ""
	}
	return e.E.Error()
}

func (e *AppError) UnmarshalJSON(b []byte) error {

	var item struct {
		Error string `json:"message"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(b, &item); err != nil {
		return err
	}

	if item.Error != "" {
		e.Code = item.Code
		e.E = fmt.Errorf(item.Error)
	}
	return nil
}

func (e *AppError) MarshalJSON() ([]byte, error) {

	if e.E != nil {
		return json.Marshal(struct {
			Error string `json:"message"`
			Code  int    `json:"code"`
		}{
			e.Error(),
			e.Code,
		})
	}
	return nil, nil
}

func (e AppError) IsRetryable() bool {
	return retryableErr[e.E]
}
