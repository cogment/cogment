// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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

//go:build cgo
// +build cgo

package wrapper

/*
	#include <stdlib.h>
	#include <time.h>

	extern void log_callback(void*, char*, int, time_t, size_t, char*, int, char*,
                                            char*);
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// Pointer to a C function able to be used as a callback for the orchestrator's logger
var cLogCallback unsafe.Pointer = C.log_callback

const (
	SpdlogLevelTrace    int = 0
	SpdlogLevelDebug    int = 1
	SpdlogLevelInfo     int = 2
	SpdlogLevelWarn     int = 3
	SpdlogLevelError    int = 4
	SpdlogLevelCritical int = 5
)

//export log_callback
func log_callback(
	_ctx unsafe.Pointer,
	_loggerName *C.char,
	level C.int,
	timestamp C.time_t,
	threadID C.size_t,
	fileName *C.char,
	lineNum C.int,
	functionName *C.char,
	message *C.char,
) {
	var logLevel logrus.Level
	// Mapping between the orchestrator levels (ie. spdlog's)
	switch int(level) {
	default:
		fallthrough
	case SpdlogLevelTrace:
		logLevel =
			logrus.TraceLevel
	case SpdlogLevelDebug:
		logLevel =
			logrus.DebugLevel
	case SpdlogLevelInfo:
		logLevel =
			logrus.InfoLevel
	case SpdlogLevelWarn:
		logLevel = logrus.WarnLevel
	case SpdlogLevelError:
		logLevel = logrus.ErrorLevel
	case SpdlogLevelCritical:
		logLevel = logrus.FatalLevel
	}
	logger := log.WithTime(time.Unix(int64(timestamp), 0)).WithFields(logrus.Fields{
		"thread": threadID,
	})

	// the location is considered empty if the line number is 0
	// cf. https://spdlog.docsforge.com/v1.x/api/spdlog/source_loc/
	if lineNum > 0 {
		logger = log.WithFields(logrus.Fields{
			"file":     C.GoString(fileName),
			"line":     lineNum,
			"function": C.GoString(functionName),
		})
	}
	logger.Log(logLevel, C.GoString(message))
}
