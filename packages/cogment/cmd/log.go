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

package cmd

import (
	"fmt"
	"os"

	"github.com/cogment/cogment/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var log = logrus.WithField("component", "cmd")

type logFormat string

const (
	text logFormat = "text"
	json logFormat = "json"
)

var expectedLogFormats = []logFormat{text, json}

func isValidLogFormat(desiredFormat logFormat) bool {
	for _, format := range expectedLogFormats {
		if format == desiredFormat {
			return true
		}
	}
	return false
}

var expectedLogLevels []string

func init() {
	expectedLogLevels = make([]string, 0)
	for _, level := range logrus.AllLevels {
		expectedLogLevels = append(expectedLogLevels, level.String())
	}
}

func configureLog(cfg *viper.Viper) error {
	// Define what is the desired log format
	desiredFormat := text // default is text
	if cfg.IsSet(servicesLogFormatKey) {
		// Explicitely specified log format
		desiredFormat = logFormat(cfg.GetString(servicesLogFormatKey))
		if !isValidLogFormat(desiredFormat) {
			return fmt.Errorf(
				"invalid log format specified %q expecting one of %v",
				desiredFormat,
				expectedLogLevels,
			)
		}
	} else if cfg.IsSet(servicesLogFileKey) {
		// default for file is json
		desiredFormat = json
	}

	// Apply the desired log format
	switch desiredFormat {
	case json:
		logrus.SetFormatter(&logrus.JSONFormatter{})
	case text:
		logrus.SetFormatter(&utils.LoggerFormatter{
			PrefixFields: []string{"component", "backend"},
		})
	}

	if cfg.IsSet(servicesLogFileKey) {
		path := cfg.GetString(servicesLogFileKey)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("unable to open log file %q: %w", path, err)
		}
		log.WithField("path", path).Info("Logger setup with a file output")
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetOutput(file)
		return nil
	}

	logLevel, err := logrus.ParseLevel(cfg.GetString(servicesLogLevelKey))
	if err != nil {
		err := fmt.Errorf(
			"invalid log level specified %q expecting one of %v",
			cfg.GetString(servicesLogLevelKey),
			expectedLogLevels,
		)
		log.WithField("error", err).Error("Unable to configure logging")
		return err
	}
	logrus.SetLevel(logLevel)

	return nil
}
