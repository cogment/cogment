// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/golang/gddo/httputil/header"
)

// HTTPError represents an http error that can be returned by the REST API
type HTTPError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// NewHTTPError creates a structured error encapsulating an http error
func NewHTTPError(status int, message string, cause error) *HTTPError {
	return &HTTPError{
		Status:  status,
		Message: message,
		Cause:   cause,
	}
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %s", e.Status, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Status, e.Message)
}

func errorHandler(handler func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		if err != nil {
			httpErr := &HTTPError{}
			if !errors.As(err, &httpErr) {
				httpErr = NewHTTPError(http.StatusInternalServerError, "Unexpected Error", err)
			}
			if httpErr.Status >= 500 {
				// Internal errors
				log.Println(err)
			}
			w.WriteHeader(httpErr.Status)
			err := encodeJSONResponse(w, &httpErr)
			if err != nil {
				// This one can't really be recovered from
				log.Println(err)
			}
		}
	}
}

func encodeJSONResponse(w http.ResponseWriter, src interface{}) error {
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(src); err != nil {
		return NewHTTPError(http.StatusInternalServerError, "unable to encode response to JSON", err)
	}

	return nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			return NewHTTPError(http.StatusUnsupportedMediaType, "Content-Type header is not application/json", nil)
		}
	}

	if r.Body == nil {
		return NewHTTPError(http.StatusBadRequest, "Missing request body", nil)
	}

	// Use http.MaxBytesReader to enforce a maximum read of 1MB from the response body.
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset), err)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return NewHTTPError(http.StatusBadRequest, "Request body contains badly-formed JSON", err)

		case errors.As(err, &unmarshalTypeError):
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset), err)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Request body contains unknown field %s", fieldName), err)

		case errors.Is(err, io.EOF):
			return NewHTTPError(http.StatusBadRequest, "Request body must not be empty", err)

		case err.Error() == "http: request body too large":
			return NewHTTPError(http.StatusRequestEntityTooLarge, "Request body must not be larger than 1MB", err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return NewHTTPError(http.StatusBadRequest, "Request body must only contain a single JSON object", err)
	}

	return nil
}

func decodeBlobBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/octet-stream" {
			return nil, NewHTTPError(http.StatusUnsupportedMediaType, "Content-Type header is not application/octet-stream", nil)
		}
	}

	if r.Body == nil {
		return nil, NewHTTPError(http.StatusBadRequest, "Missing request body", nil)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func encodeBlobResponse(w http.ResponseWriter, src []byte) error {
	srcSize := len(src)
	w.Header().Add("Content-Type", "application/octet-stream")

	size, err := w.Write(src)
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to write given blob of size %d in response, only %d bytes written", srcSize, size), nil)
	}
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("unable to write given blob of size %d in response", srcSize), err)
	}

	return nil
}
