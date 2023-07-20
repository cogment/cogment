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

package launcher

import (
	"github.com/cogment/cogment/utils"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func configureLog(quietLevel int) {
	prefixFields := []string{"cmd"}

	levelNames := map[logrus.Level]string{
		logrus.TraceLevel: "TRACE ",
		logrus.DebugLevel: "INFO  ", // Repurposed
		logrus.InfoLevel:  "stdout", // We appropriate this level for the stdout process output
		logrus.WarnLevel:  "stderr", // We appropriate this level for the stderr process output
		logrus.ErrorLevel: "ERROR ", // Unused
		logrus.FatalLevel: "FATAL ", // Unused
		logrus.PanicLevel: "PANIC ", // Unused
	}

	minimalOutput := (quietLevel > 2)

	loggerFormatter := utils.MakeLoggerFormatter(prefixFields, levelNames, minimalOutput)
	log.SetFormatter(&loggerFormatter)

	if quietLevel <= 0 {
		log.SetLevel(logrus.TraceLevel)
	} else if quietLevel == 1 {
		log.SetLevel(logrus.DebugLevel)
	} else if quietLevel >= 2 {
		log.SetLevel(logrus.InfoLevel)
	}
}
