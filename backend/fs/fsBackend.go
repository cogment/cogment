// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

package fs

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"regexp"
	"text/template"
	"time"

	"github.com/cogment/model-registry/backend"
	"gopkg.in/yaml.v2"
)

type fsBackend struct {
	rootDirname string
}

var versionDataFilenameTemplate = template.Must(template.New("versionDataFilenameTemplate").Parse("{{ .ModelID }}-v{{ .Number | printf \"%06d\" }}.data"))

var versionInfoFilenameTemplate = template.Must(template.New("versionInfoFilenameTemplate").Parse("{{ .ModelID }}-v{{ .Number | printf \"%06d\" }}.yaml"))
var versionInfoFilenameRegexp = regexp.MustCompile("[a-zA-Z][a-zA-Z0-9-_]*-v[0-9]+.yaml")

var modelDirnameRegexp = regexp.MustCompile("[a-zA-Z][a-zA-Z0-9-_]*")

// CreateBackend creates a new backend using the local filesystem
func CreateBackend(rootDirname string) (backend.Backend, error) {
	rootDirentry, err := os.Stat(rootDirname)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("Unable to create filesystem backend: %q doesn't exist", rootDirname)
	}
	if !rootDirentry.IsDir() {
		return nil, fmt.Errorf("Unable to create filesystem backend: %q is not a directory", rootDirname)
	}
	backend := fsBackend{
		rootDirname: rootDirname,
	}
	return &backend, nil
}

// Destroy terminates the underlying storage
func (b *fsBackend) Destroy() {
	// Nothing
}

// CreateModel creates a model with a given unique id in the backend
func (b *fsBackend) CreateModel(modelID string) error {
	modelDirname := path.Join(b.rootDirname, modelID)
	_, err := os.Stat(modelDirname)
	if !os.IsNotExist(err) {
		return &backend.ModelAlreadyExistsError{ModelID: modelID}
	}

	err = os.Mkdir(modelDirname, 0750)
	if err != nil {
		return fmt.Errorf("unable to create model %q: %w", modelID, err)
	}

	return nil
}

func loadVersionInfoFile(versionInfoFilename string) (backend.VersionInfo, error) {
	versionInfoData, err := os.ReadFile(versionInfoFilename)
	if err != nil {
		return backend.VersionInfo{}, fmt.Errorf("unable to read version info from %q: %w", versionInfoFilename, err)
	}

	versionInfo := backend.VersionInfo{}
	err = yaml.Unmarshal(versionInfoData, &versionInfo)
	if err != nil {
		return backend.VersionInfo{}, fmt.Errorf("unable to deserialize version info from %q: %w", versionInfoFilename, err)
	}

	return versionInfo, nil
}

func (b *fsBackend) retrieveModelLatestVersionInfo(modelID string) (backend.VersionInfo, error) {
	modelDirname := path.Join(b.rootDirname, modelID)
	modelDirContent, err := os.ReadDir(modelDirname)
	if err != nil {
		return backend.VersionInfo{}, &backend.UnknownModelError{ModelID: modelID}
	}

	for i := len(modelDirContent) - 1; i >= 0; i-- {
		entry := modelDirContent[i]
		if !entry.IsDir() && versionInfoFilenameRegexp.MatchString(entry.Name()) {
			// This is the latest version info file
			latestVersionInfoFilename := path.Join(modelDirname, entry.Name())
			latestVersionInfo, err := loadVersionInfoFile(latestVersionInfoFilename)
			if err != nil {
				return backend.VersionInfo{}, fmt.Errorf("unable to retrieve model %q latest version info: %w", modelID, err)
			}
			return latestVersionInfo, nil
		}
	}
	// No versions
	return backend.VersionInfo{}, nil
}

// HasModel check if a model exists
func (b *fsBackend) HasModel(modelID string) (bool, error) {
	modelDirname := path.Join(b.rootDirname, modelID)
	_, err := os.Stat(modelDirname)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, err
}

// DeleteModel deletes a model with a given id from the storage
func (b *fsBackend) DeleteModel(modelID string) error {
	_, err := b.retrieveModelLatestVersionInfo(modelID)
	if err != nil {
		return err
	}

	modelDirname := path.Join(b.rootDirname, modelID)
	err = os.RemoveAll(modelDirname)
	if err != nil {
		return fmt.Errorf("unable to delete model %q: %w", modelID, err)
	}

	return nil
}

func filteredReadDir(dirname string, offset int, limit int, filter func(fs.DirEntry) bool) ([]fs.DirEntry, error) {
	dirContent, err := os.ReadDir(dirname)
	if err != nil {
		return []fs.DirEntry{}, err
	}

	filteredEntryIdx := 0
	filteredEntries := []fs.DirEntry{}
	from := 0
	if offset >= 0 {
		from = offset
	}
	to := len(dirContent)
	if limit >= 0 {
		to = from + limit
	}
	for _, entry := range dirContent {
		if filter(entry) {
			if filteredEntryIdx < from {
				filteredEntryIdx++
				continue
			}
			if filteredEntryIdx >= to {
				break
			}
			filteredEntries = append(filteredEntries, entry)
			filteredEntryIdx++
		}
	}
	return filteredEntries, nil
}

// ListModels list models ordered by id from the given offset index, it returns at most the given limit number of models
func (b *fsBackend) ListModels(offset int, limit int) ([]string, error) {
	modelEntries, err := filteredReadDir(b.rootDirname, offset, limit, func(entry fs.DirEntry) bool {
		return entry.IsDir() && modelDirnameRegexp.MatchString(entry.Name())
	})
	if err != nil {
		return []string{}, fmt.Errorf("unable to list models: %w", err)
	}

	models := []string{}
	for _, entry := range modelEntries {
		models = append(models, entry.Name())
	}

	return models, nil
}

func (b *fsBackend) buildVersionInfoFilename(versionInfo backend.VersionInfo) string {
	versionInfoFilenameBuffer := new(bytes.Buffer)
	err := versionInfoFilenameTemplate.Execute(versionInfoFilenameBuffer, versionInfo)
	if err != nil {
		panic(err)
	}
	return path.Join(b.rootDirname, versionInfo.ModelID, versionInfoFilenameBuffer.String())
}

func (b *fsBackend) buildVersionDataFilename(versionInfo backend.VersionInfo) string {
	versionDataFilenameBuffer := new(bytes.Buffer)
	err := versionDataFilenameTemplate.Execute(versionDataFilenameBuffer, versionInfo)
	if err != nil {
		panic(err)
	}
	return path.Join(b.rootDirname, versionInfo.ModelID, versionDataFilenameBuffer.String())
}

// CreateModelVersion creates and store a new version for a model and returns its info, including the version number
func (b *fsBackend) CreateOrUpdateModelVersion(modelID string, versionNumber int, data []byte, archive bool) (backend.VersionInfo, error) {
	var versionInfo backend.VersionInfo
	if versionNumber <= 0 {
		// Create a new version after the last one
		latestVersionInfo, err := b.retrieveModelLatestVersionInfo(modelID)
		if err != nil {
			return backend.VersionInfo{}, err
		}
		versionInfo = backend.VersionInfo{
			ModelID:   modelID,
			CreatedAt: time.Now(),
			Archive:   archive,
			Hash:      backend.ComputeHash(data),
			Number:    latestVersionInfo.Number + 1,
		}
	} else {
		// Maybe there is an existing version
		existingVersionInfo, err := b.RetrieveModelVersionInfo(modelID, versionNumber)
		if err != nil {
			if _, ok := err.(*backend.UnknownModelVersionError); !ok {
				return backend.VersionInfo{}, err
			}
			// No version, create a new one
			versionInfo = backend.VersionInfo{
				ModelID:   modelID,
				CreatedAt: time.Now(),
				Archive:   archive,
				Hash:      backend.ComputeHash(data),
				Number:    versionNumber,
			}
		} else {
			// Update an existing version
			versionInfo = existingVersionInfo
			versionInfo.Archive = archive
			versionInfo.Hash = backend.ComputeHash(data)
		}
	}

	versionDataFilename := b.buildVersionDataFilename(versionInfo)

	err := os.WriteFile(versionDataFilename, data, 0640)
	if err != nil {
		return backend.VersionInfo{}, fmt.Errorf("unable to create a version for model %q: %w", modelID, err)
	}

	versionInfoFilename := b.buildVersionInfoFilename(versionInfo)

	versionInfoData, err := yaml.Marshal(versionInfo)
	if err != nil {
		os.Remove(versionDataFilename)
		return backend.VersionInfo{}, fmt.Errorf("unable to create a version for model %q: %w", modelID, err)
	}

	err = os.WriteFile(versionInfoFilename, versionInfoData, 0640)
	if err != nil {
		os.Remove(versionDataFilename)
		return backend.VersionInfo{}, fmt.Errorf("unable to create a version for model %q: %w", modelID, err)
	}

	return versionInfo, nil
}

// RetrieveModelVersionInfo retrieves a given model version info
func (b *fsBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	if versionNumber <= 0 {
		// Retrieve the latest version
		latestVersionInfo, err := b.retrieveModelLatestVersionInfo(modelID)
		if err != nil {
			return backend.VersionInfo{}, err
		}
		if latestVersionInfo.Number == 0 {
			return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: -1}
		}
		return latestVersionInfo, nil
	}
	// Retrieve a specific version
	versionInfoFilename := b.buildVersionInfoFilename(backend.VersionInfo{ModelID: modelID, Number: versionNumber})
	_, err := os.Stat(versionInfoFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return backend.VersionInfo{}, fmt.Errorf("unable to read version info from %q: %w", versionInfoFilename, err)
	}
	versionInfo, err := loadVersionInfoFile(versionInfoFilename)
	if err != nil {
		return backend.VersionInfo{}, err
	}
	return versionInfo, err
}

// RetrieveModelVersion retrieves a given model version data
func (b *fsBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	versionInfo := backend.VersionInfo{
		ModelID: modelID,
		Number:  versionNumber,
	}
	if versionNumber <= 0 {
		// Retrieve the latest version
		var err error
		latestVersionInfo, err := b.retrieveModelLatestVersionInfo(modelID)
		if err != nil {
			return []byte{}, err
		}
		if latestVersionInfo.Number == 0 {
			return []byte{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: -1}
		}
		versionInfo = latestVersionInfo
	}
	// Retrieve a specific version
	versionDataFilename := b.buildVersionDataFilename(versionInfo)
	versionData, err := os.ReadFile(versionDataFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return []byte{}, fmt.Errorf("unable to read data for model %q version \"%d\": %w", versionInfo.ModelID, versionInfo.Number, err)
	}
	return versionData, nil
}

// DeleteModelVersion deletes a given model version
func (b *fsBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	versionInfo := backend.VersionInfo{
		ModelID: modelID,
		Number:  versionNumber,
	}
	versionInfoFilename := b.buildVersionInfoFilename(versionInfo)
	err := os.Remove(versionInfoFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return fmt.Errorf("unable to delete model %q version \"%d\" info: %w", modelID, versionNumber, err)
	}
	versionDataFilename := b.buildVersionDataFilename(versionInfo)
	err = os.Remove(versionDataFilename)
	if err != nil {
		return fmt.Errorf("unable to delete model %q version \"%d\" data: %w", modelID, versionNumber, err)
	}
	return nil
}

// ListModelVersionInfos list the versions info of a model from the latest to the earliest from the given offset index, it returns at most the given limit number of versions
func (b *fsBackend) ListModelVersionInfos(modelID string, offset int, limit int) ([]backend.VersionInfo, error) {
	modelDirname := path.Join(b.rootDirname, modelID)
	modelVersionEntries, err := filteredReadDir(modelDirname, offset, limit, func(entry fs.DirEntry) bool {
		return !entry.IsDir() && versionInfoFilenameRegexp.MatchString(entry.Name())
	})
	if err != nil {
		return []backend.VersionInfo{}, &backend.UnknownModelError{ModelID: modelID}
	}

	versions := []backend.VersionInfo{}
	for _, entry := range modelVersionEntries {
		versionInfoFilename := path.Join(modelDirname, entry.Name())
		versionInfo, err := loadVersionInfoFile(versionInfoFilename)
		if err != nil {
			log.Printf("unable to unmarshall model version info from %q, skipping the version: %s", versionInfoFilename, err)
			continue
		}
		versions = append(versions, versionInfo)
	}
	return versions, nil
}
