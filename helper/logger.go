// Copyright 2021 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

package helper

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

var baseLogger *zap.Logger
var logger *zap.SugaredLogger

func CheckError(err error) {
	if err != nil {
		logger.Fatal(err)
	}
}

func CheckErrorf(err error, template string, args ...interface{}) {
	if err != nil {
		errFormatStr := fmt.Sprintf("%s: %v", template, err)
		logger.Fatalf(errFormatStr, args...)
	}
}

func GetLogger(names []string) *zap.Logger {
	var newLogger = baseLogger.WithOptions(zap.AddStacktrace(zap.WarnLevel), zap.AddCaller(), zap.AddCallerSkip(1))
	for _, name := range names {
		newLogger = newLogger.Named(name)
	}

	return newLogger
}

func GetSugarLogger(names []string) *zap.SugaredLogger {
	return GetLogger(names).Sugar()
}

func init() {
	// TODO: determine runtime environment
	var err error
	baseLogger, err = zap.NewDevelopment()
	if err != nil {
		err = fmt.Errorf("error instantiating logger: %v", err)
		fmt.Println(err)
		os.Exit(1)
	}
	baseLogger = baseLogger.Named("cogment-cli")

	logger = GetSugarLogger([]string{"helper"})
}
