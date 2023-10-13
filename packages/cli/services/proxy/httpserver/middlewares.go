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
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func ginLoggerMiddleware(c *gin.Context) {
	method := c.Request.Method
	path := c.Request.URL.Path

	start := time.Now()
	c.Next()
	stop := time.Since(start)

	statusCode := c.Writer.Status()
	dataLength := c.Writer.Size()
	if dataLength < 0 {
		dataLength = 0
	}

	entry := log.WithFields(logrus.Fields{
		"statusCode": statusCode,
		"latency":    int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0)),
		"clientIP":   c.ClientIP(),
		"referer":    c.Request.Referer(),
		"dataLength": dataLength,
		"userAgent":  c.Request.UserAgent(),
	})

	if statusCode >= http.StatusInternalServerError {
		entry.Errorf("[%s] [%s] - 5XX internal error", method, path)
	} else if statusCode >= http.StatusBadRequest {
		entry.Warnf("[%s] [%s] - 4XX request error", method, path)
	} else {
		entry.Debugf("[%s] [%s]", method, path)
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

	if len(c.Errors) > 0 {
		ret := gin.H{
			"message": c.Errors.Last().Error(),
		}
		if len(c.Errors) > 1 {
			ret["errors"] = c.Errors
		}

		c.JSON(statusCode, ret)
	}
}
