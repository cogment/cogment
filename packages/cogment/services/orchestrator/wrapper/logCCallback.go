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
var c_log_callback unsafe.Pointer = C.log_callback

//export log_callback
func log_callback(_ctx unsafe.Pointer, _loggerName *C.char, level C.int, timestamp C.time_t, threadId C.size_t, fileName *C.char, lineNum C.int, functionName *C.char, message *C.char) {
	var logLevel logrus.Level
	// Mapping between the orchestrator levels (ie. spdlog's)
	switch level {
	default:
		fallthrough
	case 0:
		logLevel =
			logrus.TraceLevel
	case 1:
		logLevel =
			logrus.DebugLevel
	case 2:
		logLevel =
			logrus.InfoLevel
	case 3:
		logLevel = logrus.WarnLevel
	case 4:
		fallthrough
	case 5:
		logLevel = logrus.ErrorLevel
	}
	logger := log.WithTime(time.Unix(int64(timestamp), 0)).WithFields(logrus.Fields{
		"thread": threadId,
	})

	// cf. https://spdlog.docsforge.com/v1.x/api/spdlog/source_loc/ the location is considered empty if the line number is 0
	if lineNum > 0 {
		logger = log.WithFields(logrus.Fields{
			"file":     C.GoString(fileName),
			"line":     lineNum,
			"function": C.GoString(functionName),
		})
	}
	logger.Log(logLevel, C.GoString(message))
}
