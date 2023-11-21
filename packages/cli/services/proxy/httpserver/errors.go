// Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"
)

type httpError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message" description:"Human-readable error description"`
	Err        error  `json:"-"`
}

func (e httpError) Error() string {
	return e.Message
}

func (e httpError) Unwrap() error {
	return e.Err
}

func wrapError(statusCode int, err error) error {
	return httpError{
		StatusCode: statusCode,
		Message:    err.Error(),
		Err:        err,
	}
}

func ginErrorHandlerMiddleware(c *gin.Context) {
	c.Next()

	statusCode := c.Writer.Status()
	log := log.WithField("status", statusCode)

	for errIndex, err := range c.Errors {
		if statusCode >= http.StatusInternalServerError {
			log.Errorf("Error #%02d - %s", errIndex+1, err)
		} else if statusCode >= http.StatusBadRequest {
			log.Debugf("Error #%02d - %s", errIndex+1, err)
		}
	}
}

func tonicErrorHook(_ *gin.Context, err error) (int, interface{}) {
	if _, ok := err.(tonic.BindError); ok {
		return http.StatusBadRequest, wrapError(http.StatusBadRequest, err)
	}
	if _, ok := err.(httpError); ok {
		return err.(httpError).StatusCode, err.(httpError)
	}
	return http.StatusInternalServerError, wrapError(http.StatusInternalServerError, err)
}
