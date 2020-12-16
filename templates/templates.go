// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
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

package templates

import (
	"os"
	"path/filepath"
	"text/template"

	"github.com/markbates/pkger"
	"gitlab.com/cogment/cogment/helper"
)

// This function is only there to let pkger static analysis knows we want to embed the templates file in the binary.
func includeTemplates() {
	pkger.Include("/templates")
}

// GenerateFromTemplate generates a file from a given template and configuration
func GenerateFromTemplate(tmplPath string, config interface{}, outputPath string) error {
	tmplFile, err := pkger.Open(tmplPath)
	if err != nil {
		return err
	}
	defer tmplFile.Close()

	tmplFileStats, err := tmplFile.Stat()
	if err != nil {
		return err
	}

	tmplFileContent := make([]byte, tmplFileStats.Size())
	tmplFile.Read(tmplFileContent)

	t := template.New(outputPath).Funcs(template.FuncMap{
		"snakeify":  helper.Snakeify,
		"kebabify":  helper.Kebabify,
		"pascalify": helper.Pascalify,
		"tocaps":    helper.Tocaps,
	})

	t = template.Must(t.Parse(string(tmplFileContent)))

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	if err = t.Execute(outputFile, config); err != nil {
		return err
	}

	return err
}
