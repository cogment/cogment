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
	// PrefixFields - the fields that will appear in the prefix in this order
	PrefixFields []string
}

// Format an log entry
func (f *LoggerFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	levelColor := getColorByLevel(entry.Level)

	// split the fields between the prefix and the rest
	prefixFields, otherFields := f.splitFields(entry)

	// output buffer
	b := &bytes.Buffer{}

	// write time
	b.WriteString(entry.Time.Format(time.RFC3339))

	// write caller information
	if entry.HasCaller() {
		fmt.Fprintf(
			b,
			" (%s:%d %s)",
			entry.Caller.File,
			entry.Caller.Line,
			entry.Caller.Function,
		)
	}

	// write level
	level := strings.ToUpper(entry.Level.String())
	fmt.Fprintf(b, " \x1b[%dm[%s]", levelColor, level[:4])

	// write prefix fields
	b.WriteString(" [")
	for fieldIdx, field := range prefixFields {
		if fieldIdx < len(prefixFields)-1 {
			fmt.Fprintf(b, "%v>", entry.Data[field])
		} else {
			fmt.Fprintf(b, "%v", entry.Data[field])
		}
	}
	b.WriteString("] \x1b[0m")

	// write message
	b.WriteString(strings.TrimSpace(entry.Message))

	// write the other fields
	for _, field := range otherFields {
		fmt.Fprintf(b, " [%s:%v]", field, entry.Data[field])
	}

	b.WriteByte('\n')

	return b.Bytes(), nil
}

func (f *LoggerFormatter) splitFields(entry *logrus.Entry) ([]string, []string) {
	prefixFields := []string{}
	otherFields := []string{}
	isPrefixField := map[string]bool{}
	for _, field := range f.PrefixFields {
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
	case logrus.WarnLevel:
		return colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return colorRed
	default:
		return colorBlue
	}
}
