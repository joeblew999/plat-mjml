package errorx

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// CodeError is a typed error that carries an HTTP status code.
// Logic functions return these so the global error handler can map
// them to the correct HTTP response.
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *CodeError) Error() string {
	return e.Msg
}

// ErrNotFound returns a 404 error.
func ErrNotFound(msg string) error {
	return &CodeError{Code: http.StatusNotFound, Msg: msg}
}

// ErrBadRequest returns a 400 error.
func ErrBadRequest(msg string) error {
	return &CodeError{Code: http.StatusBadRequest, Msg: msg}
}

// ErrInternal returns a 500 error.
func ErrInternal(msg string) error {
	return &CodeError{Code: http.StatusInternalServerError, Msg: msg}
}

// RegisterErrorHandler installs a global error handler that maps CodeError
// to the correct HTTP status code. Untyped errors become 500.
func RegisterErrorHandler() {
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		switch e := err.(type) {
		case *CodeError:
			return e.Code, &CodeError{Code: e.Code, Msg: e.Msg}
		default:
			logx.WithContext(ctx).Errorf("unexpected error: %v", err)
			return http.StatusInternalServerError, &CodeError{
				Code: http.StatusInternalServerError,
				Msg:  "internal server error",
			}
		}
	})
}
