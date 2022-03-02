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

func configureLog(cfg *viper.Viper) error {
	// Formatter
	logrus.SetFormatter(&utils.LoggerFormatter{
		PrefixFields: []string{"component", "backend"},
	})

	if servicesViper.IsSet(servicesLogFileKey) {
		path := servicesViper.GetString(servicesLogFileKey)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("unable to open log file %q: %w", path, err)
		}
		log.WithField("path", path).Info("Logger setup with a file output")
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetOutput(file)
		return nil
	}

	logLevel, err := logrus.ParseLevel(servicesViper.GetString(servicesLogLevelKey))
	if err != nil {
		err := fmt.Errorf(
			"invalid log level specified %q expecting one of %v",
			servicesViper.GetString(servicesLogLevelKey),
			expectedLogLevels,
		)
		log.WithField("error", err).Error("Unable to configure logging")
		return err
	}
	logrus.SetLevel(logLevel)

	return nil
}
