package httputils

import (
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/versions"
	"github.com/gorilla/mux"
)

// httpStatusError is an interface
// that errors with custom status codes
// implement to tell the api layer
// which response status to set.
type httpStatusError interface {
	HTTPErrorStatusCode() int
}

// inputValidationError is an interface
// that errors generated by invalid
// inputs can implement to tell the
// api layer to set a 400 status code
// in the response.
type inputValidationError interface {
	IsValidationError() bool
}

// GetHTTPErrorStatusCode retrieve status code from error message
func GetHTTPErrorStatusCode(err error) int {
	if err == nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error("unexpected HTTP error handling")
		return http.StatusInternalServerError
	}

	var statusCode int
	errMsg := err.Error()

	switch e := err.(type) {
	case httpStatusError:
		statusCode = e.HTTPErrorStatusCode()
	case inputValidationError:
		statusCode = http.StatusBadRequest
	default:
		// FIXME: this is brittle and should not be necessary, but we still need to identify if
		// there are errors falling back into this logic.
		// If we need to differentiate between different possible error types,
		// we should create appropriate error types that implement the httpStatusError interface.
		errStr := strings.ToLower(errMsg)
		for keyword, status := range map[string]int{
			"not found":             http.StatusNotFound,
			"no such":               http.StatusNotFound,
			"bad parameter":         http.StatusBadRequest,
			"no command":            http.StatusBadRequest,
			"conflict":              http.StatusConflict,
			"impossible":            http.StatusNotAcceptable,
			"wrong login/password":  http.StatusUnauthorized,
			"unauthorized":          http.StatusUnauthorized,
			"hasn't been activated": http.StatusForbidden,
		} {
			if strings.Contains(errStr, keyword) {
				statusCode = status
				break
			}
		}
	}

	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	return statusCode
}

// MakeErrorHandler makes an HTTP handler that decodes a Docker error and
// returns it in the response.
func MakeErrorHandler(err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusCode := GetHTTPErrorStatusCode(err)
		vars := mux.Vars(r)
		if vars["version"] == "" || versions.GreaterThan(vars["version"], "1.23") {
			response := &types.ErrorResponse{
				Message: err.Error(),
			}
			WriteJSON(w, statusCode, response)
		} else {
			http.Error(w, err.Error(), statusCode)
		}
	}
}
