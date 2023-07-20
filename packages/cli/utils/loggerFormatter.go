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

package utils

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LoggerFormatter struct {
	// The fields that will appear in the prefix in this order
	PrefixFields []string

	// The names output for each log level
	LevelNames map[logrus.Level]string

	// Whether to output only the minimum: no prefix fields, no time, no level, no color
	Minimal bool
}

func MakeLoggerFormatter(prefix []string, levelNames map[logrus.Level]string, minimal bool) LoggerFormatter {
	var logger LoggerFormatter
	logger.Minimal = minimal

	if prefix != nil {
		logger.PrefixFields = prefix
	} else {
		logger.PrefixFields = []string{}
	}

	if levelNames != nil {
		logger.LevelNames = levelNames
	} else {
		logger.LevelNames = map[logrus.Level]string{
			logrus.TraceLevel: "TRAC",
			logrus.DebugLevel: "DEBU",
			logrus.InfoLevel:  "INFO",
			logrus.WarnLevel:  "WARN",
			logrus.ErrorLevel: "ERRO",
			logrus.FatalLevel: "FATA",
			logrus.PanicLevel: "PANI",
		}
	}

	return logger
}

// Format an log entry
func (formatter *LoggerFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	outBuf := &bytes.Buffer{}

	// split the fields between the prefix and the rest
	prefixFields, otherFields := formatter.splitFields(entry)

	if !formatter.Minimal {
		// write time
		outBuf.WriteString(entry.Time.Format(time.RFC3339))

		// write caller information
		if entry.HasCaller() {
			fmt.Fprintf(
				outBuf,
				" (%s:%d %s)",
				entry.Caller.File,
				entry.Caller.Line,
				entry.Caller.Function,
			)
		}
		outBuf.WriteString(" ")

		// set level color
		levelColor := getColorByLevel(entry.Level)
		fmt.Fprintf(outBuf, "\x1b[%dm", levelColor)

		// write level
		level, ok := formatter.LevelNames[entry.Level]
		if ok {
			fmt.Fprintf(outBuf, "[%s] ", level)
		}

		// write prefix fields
		outBuf.WriteString("[")
		for fieldIdx, field := range prefixFields {
			if fieldIdx < len(prefixFields)-1 {
				fmt.Fprintf(outBuf, "%v>", entry.Data[field])
			} else {
				fmt.Fprintf(outBuf, "%v", entry.Data[field])
			}
		}
		outBuf.WriteString("]")

		// set color back to default
		outBuf.WriteString("\x1b[0m")

		outBuf.WriteString(" ")
	}

	// write message
	outBuf.WriteString(strings.TrimSpace(entry.Message))

	// write the other fields
	for _, field := range otherFields {
		if len(field) != 0 {
			fmt.Fprintf(outBuf, " [%s:%v]", field, entry.Data[field])
		} else {
			fmt.Fprintf(outBuf, " [%v]", entry.Data[field])
		}
	}

	outBuf.WriteByte('\n')

	return outBuf.Bytes(), nil
}

func (formatter *LoggerFormatter) splitFields(entry *logrus.Entry) ([]string, []string) {
	prefixFields := []string{}
	otherFields := []string{}
	isPrefixField := map[string]bool{}
	for _, field := range formatter.PrefixFields {
		isPrefixField[field] = true
		if _, ok := entry.Data[field]; ok {
			prefixFields = append(prefixFields, field)
		}
	}
	for field := range entry.Data {
		if _, ok := isPrefixField[field]; !ok {
			otherFields = append(otherFields, field)
		}
	}
	sort.Strings(otherFields)
	return prefixFields, otherFields
}

const (
	colorRed    = 31
	colorYellow = 33
	colorBlue   = 36
	colorGray   = 37
)

func getColorByLevel(level logrus.Level) int {
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return colorGray
	case logrus.InfoLevel:
		return colorBlue
	case logrus.WarnLevel:
		return colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return colorRed
	default:
		return colorBlue
	}
}
