// Package api provides functions for the generated API.
package api

import (
	"encoding/json"
	"net/http"
)

// SendError sends an error response.
func SendError(writer http.ResponseWriter, httpStatus int, errorType string, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(httpStatus)

	if err := json.NewEncoder(writer).Encode(GenericErrorResponse{
		Error:   errorType,
		Message: message,
	}); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	}
}
