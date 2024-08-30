package pderr

import (
	"fmt"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PDError interface {
	Error() string
	Unwrap() error
	GRPCStatus() *status.Status
	ErrorCode() codes.Code
}

type wrappedGRPCClientError struct {
	method string
	err    error
}

func (g wrappedGRPCClientError) Error() string {
	return fmt.Sprintf("calling gRPC method %s: %s", g.method, g.err.Error())
}

func (g wrappedGRPCClientError) ErrorCode() codes.Code {
	st, ok := status.FromError(g.err)
	if !ok {
		return codes.Unknown
	}

	return st.Code()
}

func (g wrappedGRPCClientError) Unwrap() error {
	return g.err
}

func (g wrappedGRPCClientError) GRPCStatus() *status.Status {
	st, _ := status.FromError(g.err)
	return st
}

type genericError struct {
	code       codes.Code
	message    string
	wrappedErr error
}

func (g genericError) Error() string {
	return g.message
}

func (g genericError) ErrorCode() codes.Code {
	return g.code
}

func (g genericError) Unwrap() error {
	return g.wrappedErr
}

func (g genericError) GRPCStatus() *status.Status {
	return status.New(g.ErrorCode(), g.Error())
}

func AsPDError(err error) PDError {
	if err == nil {
		return nil
	}

	if rv, ok := err.(PDError); ok {
		return rv
	}

	return genericError{
		code:       codes.Unknown,
		message:    err.Error(),
		wrappedErr: err,
	}
}

func Error(code codes.Code, message string) error {
	return genericError{
		code:       code,
		message:    message,
		wrappedErr: nil,
	}
}

func CodeOf(err error) codes.Code {
	if err == nil {
		return codes.OK
	}
	code := AsPDError(err).ErrorCode()
	return code
}

func UnknownError(message string) error {
	return Error(codes.Unknown, message)
}

func NotImplemented(message string) error {
	return Error(codes.Unimplemented, message)
}

func Unexpectedf(format string, args ...any) error {
	return AsPDError(fmt.Errorf(format, args...))
}

func BadInput(message string, inputName string, inputValue string) error {
	return Error(codes.InvalidArgument, fmt.Sprintf("bad input: %s, invalid value for %q was %q", message, inputName, inputValue))
}

func MissingRequiredFlag(flagName string) error {
	return Error(codes.InvalidArgument, fmt.Sprintf("missing required flag (%s)", flagName))
}

func WrapGRPCClient(method string, err error) error {
	return wrappedGRPCClientError{
		method: method,
		err:    err,
	}
}

func Wrap(messagePrefix string, err error) error {
	return WrapAs(CodeOf(err), messagePrefix, err)
}

func WrapAs(code codes.Code, messagePrefix string, err error) error {
	if err == nil {
		return nil
	}

	return genericError{
		code:       code,
		message:    fmt.Sprintf("%s: %s", messagePrefix, err.Error()),
		wrappedErr: err,
	}
}

func CheckOrPanic(err error) {
	if err == nil {
		return
	}

	panic(err)
}

func HandleFatalAndDie(err error) {
	if err == nil {
		os.Exit(0)
	}

	fmt.Fprintf(os.Stderr, "fatal: %s", err.Error())
	os.Exit(1)
}
