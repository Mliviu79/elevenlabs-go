package elevenlabs

import (
	"fmt"
	"net/http"
	"strings"
)

// APIError represents an error response from the API: a 400 or 401 response whose body decoded as
// the API's error. Every other unsuccessful response is a [ValidationError] or a [StatusError].
type APIError struct {
	Detail APIErrorDetail `json:"detail"`
	// StatusCode is the HTTP status of the response the error came from.
	StatusCode int `json:"-"`
}

// APIErrorDetail contains detailed information about an APIError.
type APIErrorDetail struct {
	Status         string `json:"status"`
	Message        string `json:"message"`
	AdditionalInfo string `json:"additional_info,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("api error - %s", e.Detail.Message)
}

// ValidationError represents a request validation error response from the API: a 422 response
// whose body decoded as the API's validation error.
type ValidationError struct {
	Detail *[]ValidationErrorDetailItem `json:"detail"`
	// StatusCode is the HTTP status of the response the error came from.
	StatusCode int `json:"-"`
}

type ValidationErrorDetailItem struct {
	Loc  []ValidationErrorDetailLocItem `json:"loc"`
	Msg  string                         `json:"msg"`
	Type string                         `json:"type"`
}

type ValidationErrorDetailLocItem string

func (i *ValidationErrorDetailLocItem) UnmarshalJSON(b []byte) error {
	*i = ValidationErrorDetailLocItem(strings.Trim(string(b), "\""))
	return nil
}

func (e *ValidationError) Error() string {
	if (*e).Detail != nil && len(*e.Detail) > 0 {
		return fmt.Sprintf("validation error - %s", (*e.Detail)[0].Msg)
	}
	return "validation error"
}

// StatusError is the error for an unsuccessful response the client does not return as an
// [*APIError] or a [*ValidationError]: a status whose body the client does not decode (403, 429
// and 5xx among them), or a 400, 401 or 422 whose body did not decode as the API's error. A
// caller classifies the response by StatusCode, never by the error's text.
type StatusError struct {
	// StatusCode is the HTTP status of the response the error came from.
	StatusCode int
	// Err is the body's decode failure when a 400, 401 or 422 body did not decode, nil otherwise.
	Err error
}

// Error returns the response's status, followed by the body's decode failure when there was one.
func (e *StatusError) Error() string {
	msg := fmt.Sprintf("unexpected HTTP status \"%d %s\" returned from server", e.StatusCode, http.StatusText(e.StatusCode))
	if e.Err != nil {
		return msg + ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the body's decode failure, or nil when the client did not decode the body.
func (e *StatusError) Unwrap() error { return e.Err }
