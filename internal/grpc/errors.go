package grpc

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
)

var errInvalidArgument = errors.New("invalid argument")

func mapErrorCode(err error) codes.Code {
	switch {
	case err == nil:
		return codes.OK
	case errors.Is(err, errInvalidArgument):
		return codes.InvalidArgument
	case strings.Contains(strings.ToLower(err.Error()), "not found"):
		return codes.NotFound
	default:
		return codes.Internal
	}
}
