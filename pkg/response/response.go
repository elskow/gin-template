package response

type ErrorSchema struct {
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type Response[T any] struct {
	Error  *ErrorSchema `json:"error,omitempty"`
	Output *T           `json:"output,omitempty"`
}

func Success[T any](output T) Response[T] {
	return Response[T]{
		Error:  nil,
		Output: &output,
	}
}

func Error[T any](code, message string) Response[T] {
	return Response[T]{
		Error: &ErrorSchema{
			ErrorCode:    code,
			ErrorMessage: message,
		},
		Output: nil,
	}
}

const (
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeInternalServerError = "INTERNAL_SERVER_ERROR"
	ErrCodeValidationFailed    = "VALIDATION_FAILED"
	ErrCodeInvalidCredentials  = "INVALID_CREDENTIALS"
)

type HTTPError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *HTTPError) Error() string {
	return e.Message
}
