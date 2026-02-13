package errors

import "fmt"

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

var (
	ErrConfigLoad      = "CONFIG_LOAD_ERROR"
	ErrDatabaseConnect = "DATABASE_CONNECT_ERROR"
	ErrRPConnect       = "RPC_CONNECT_ERROR"
	ErrBlockFetch      = "BLOCK_FETCH_ERROR"
	ErrEventParse      = "EVENT_PARSE_ERROR"
	ErrBalanceUpdate   = "BALANCE_UPDATE_ERROR"
	ErrPointsCalc      = "POINTS_CALCULATION_ERROR"
	ErrInvalidChain    = "INVALID_CHAIN_ERROR"
)
