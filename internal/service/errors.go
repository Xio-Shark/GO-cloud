package service

import "errors"

type ErrorCode string

const (
	ErrorCodeValidation ErrorCode = "validation"
	ErrorCodeNotFound   ErrorCode = "not_found"
	ErrorCodeConflict   ErrorCode = "conflict"
)

type AppError struct {
	Code    ErrorCode
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func ValidationError(message string) error {
	return &AppError{Code: ErrorCodeValidation, Message: message}
}

func NotFoundError(message string) error {
	return &AppError{Code: ErrorCodeNotFound, Message: message}
}

func ConflictError(message string) error {
	return &AppError{Code: ErrorCodeConflict, Message: message}
}

func HasErrorCode(err error, code ErrorCode) bool {
	var appErr *AppError
	return errors.As(err, &appErr) && appErr.Code == code
}
