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

package fileSystem

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"github.com/cogment/cogment/services/modelRegistry/backend"
	"github.com/rogpeppe/go-internal/lockedfile"
	"gopkg.in/yaml.v2"
)

type fsModelInfo struct {
	ModelID  string            `yaml:"model_id"`
	UserData map[string]string `yaml:"user_data"`
}

func saveModelInfoFile(modelInfoFilename string, modelInfo backend.ModelInfo) error {
	modelInfoData, err := yaml.Marshal(fsModelInfo{
		ModelID:  modelInfo.ModelID,
		UserData: modelInfo.UserData,
	})
	if err != nil {
		return fmt.Errorf(
			"unable to save model %q to %q: yaml serialization failed %w",
			modelInfo.ModelID, modelInfoFilename, err,
		)
	}

	modelDirname := path.Dir(modelInfoFilename)
	_, err = os.Stat(modelDirname)
	if os.IsNotExist(err) {
		err = os.Mkdir(modelDirname, 0750)
		if err != nil {
			return fmt.Errorf(
				"unable to save model %q to %q: directory creation failed %w",
				modelInfo.ModelID, modelInfoFilename, err,
			)
		}
	}

	err = lockedfile.Write(modelInfoFilename, bytes.NewReader(modelInfoData), 0640)

	if err != nil {
		return fmt.Errorf(
			"unable to save model %q to %q: writing to file failed %w",
			modelInfo.ModelID, modelInfoFilename, err,
		)
	}
	return nil
}

func loadModelInfoFile(modelInfoFilename string) (backend.ModelInfo, error) {
	modelInfoData, err := lockedfile.Read(modelInfoFilename)
	if err != nil {
		return backend.ModelInfo{}, fmt.Errorf("unable to read model info from %q: %w", modelInfoFilename, err)
	}

	modelInfo := fsModelInfo{}
	err = yaml.Unmarshal(modelInfoData, &modelInfo)
	if err != nil {
		return backend.ModelInfo{}, fmt.Errorf("unable to deserialize model info from %q: %w", modelInfoFilename, err)
	}

	return backend.ModelInfo{
		ModelID:  modelInfo.ModelID,
		UserData: modelInfo.UserData,
	}, nil
}

type fsVersionInfo struct {
	ModelID           string            `yaml:"model_id"`
	VersionNumber     uint              `yaml:"version_number"`
	Transient         bool              `yaml:"transient,omitempty"`
	CreationTimestamp time.Time         `yaml:"creation_timestamp"`
	DataHash          string            `yaml:"data_hash"`
	DataSize          int               `yaml:"data_size"`
	UserData          map[string]string `yaml:"user_data"`
}

func saveVersionInfoFile(versionInfoFilename string, versionInfo backend.VersionInfo) error {
	versionInfoData, err := yaml.Marshal(fsVersionInfo{
		ModelID:           versionInfo.ModelID,
		VersionNumber:     versionInfo.VersionNumber,
		CreationTimestamp: versionInfo.CreationTimestamp,
		Transient:         !versionInfo.Archived,
		DataHash:          versionInfo.DataHash,
		DataSize:          versionInfo.DataSize,
		UserData:          versionInfo.UserData,
	})
	if err != nil {
		return fmt.Errorf(
			"unable to save version info for model \"%s@%d\" to %q: yaml serialization failed %w",
			versionInfo.ModelID, versionInfo.VersionNumber, versionInfoFilename, err,
		)
	}

	modelDirname := path.Dir(versionInfoFilename)
	_, err = os.Stat(modelDirname)
	if os.IsNotExist(err) {
		err = os.Mkdir(modelDirname, 0750)
		if err != nil {
			return fmt.Errorf(
				"unable to save version info for model \"%s@%d\" to %q: directory creation failed %w",
				versionInfo.ModelID, versionInfo.VersionNumber, versionInfoFilename, err,
			)
		}
	}

	err = lockedfile.Write(versionInfoFilename, bytes.NewReader(versionInfoData), 0640)

	if err != nil {
		return fmt.Errorf(
			"unable to save version info for model \"%s@%d\" to %q: writing to file failed %w",
			versionInfo.ModelID, versionInfo.VersionNumber, versionInfoFilename, err,
		)
	}
	return nil
}

func loadVersionInfoFile(versionInfoFilename string) (backend.VersionInfo, error) {
	versionInfoData, err := lockedfile.Read(versionInfoFilename)
	if err != nil {
		return backend.VersionInfo{}, fmt.Errorf("unable to read version info from %q: %w", versionInfoFilename, err)
	}

	versionInfo := fsVersionInfo{}
	err = yaml.Unmarshal(versionInfoData, &versionInfo)
	if err != nil {
		return backend.VersionInfo{}, fmt.Errorf("unable to deserialize version info from %q: %w", versionInfoFilename, err)
	}

	return backend.VersionInfo{
		ModelID:           versionInfo.ModelID,
		VersionNumber:     versionInfo.VersionNumber,
		CreationTimestamp: versionInfo.CreationTimestamp,
		DataHash:          versionInfo.DataHash,
		DataSize:          versionInfo.DataSize,
		Archived:          !versionInfo.Transient,
		UserData:          versionInfo.UserData,
	}, nil
}

type fsBackend struct {
	rootDirname string
}

var versionDataFilenameTemplate = template.Must(
	template.New("versionDataFilenameTemplate").Parse(`{{ .ModelID }}-v{{ .VersionNumber | printf "%06d" }}.data`),
)

var versionInfoFilenameTemplate = template.Must(
	template.New("versionInfoFilenameTemplate").Parse(`{{ .ModelID }}-v{{ .VersionNumber | printf "%06d" }}.yaml`),
)
var modelInfoFilenameTemplate = template.Must(
	template.New("modelInfoFilenameTemplate").Parse(`{{ .ModelID }}.yaml`),
)
var versionInfoFilenameRegexp = regexp.MustCompile("([a-zA-Z][a-zA-Z0-9-_]*)-v([0-9]+).yaml")

var modelDirnameRegexp = regexp.MustCompile("([a-zA-Z][a-zA-Z0-9-_]*)")

// CreateBackend creates a new backend using the local filesystem
func CreateBackend(rootDirname string) (backend.Backend, error) {
	rootDirentry, err := os.Stat(rootDirname)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to create filesystem backend: %q doesn't exist", rootDirname)
	}
	if !rootDirentry.IsDir() {
		return nil, fmt.Errorf("unable to create filesystem backend: %q is not a directory", rootDirname)
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

func (b *fsBackend) retrieveModelNthToLastVersionInfo(
	modelID string,
	nthToLastIndex uint,
) (backend.VersionInfo, error) {
	modelDirname := path.Join(b.rootDirname, modelID)
	modelDirContent, err := os.ReadDir(modelDirname)
	if err != nil {
		return backend.VersionInfo{}, &backend.UnknownModelError{ModelID: modelID}
	}

	nthToLastNextIndex := uint(0)
	for i := len(modelDirContent) - 1; i >= 0; i-- {
		entry := modelDirContent[i]
		if !entry.IsDir() && versionInfoFilenameRegexp.MatchString(entry.Name()) {
			if nthToLastNextIndex == nthToLastIndex {
				// We've reached the target
				latestVersionInfoFilename := path.Join(modelDirname, entry.Name())
				latestVersionInfo, err := loadVersionInfoFile(latestVersionInfoFilename)
				if err != nil {
					return backend.VersionInfo{}, fmt.Errorf("unable to retrieve model %q latest version info: %w", modelID, err)
				}
				return latestVersionInfo, nil
			}
			nthToLastNextIndex++
		}
	}
	// No versions
	return backend.VersionInfo{}, nil
}

func (b *fsBackend) CreateOrUpdateModel(modelArgs backend.ModelInfo) (backend.ModelInfo, error) {
	modelInfo := backend.ModelInfo{
		ModelID:  modelArgs.ModelID,
		UserData: modelArgs.UserData,
	}
	modelInfoFilename := b.buildModelInfoFilename(modelInfo)
	err := saveModelInfoFile(modelInfoFilename, modelInfo)
	if err != nil {
		return backend.ModelInfo{}, err
	}

	return modelInfo, nil
}

func (b *fsBackend) RetrieveModelInfo(modelID string) (backend.ModelInfo, error) {
	modelInfoFilename := b.buildModelInfoFilename(backend.ModelInfo{ModelID: modelID})

	_, err := os.Stat(modelInfoFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return backend.ModelInfo{}, &backend.UnknownModelError{ModelID: modelID}
		}
		return backend.ModelInfo{}, fmt.Errorf("unable to read model info from %q: %w", modelInfoFilename, err)
	}

	modelInfo, err := loadModelInfoFile(modelInfoFilename)
	if err != nil {
		return backend.ModelInfo{}, err
	}

	return modelInfo, nil
}

func (b *fsBackend) RetrieveModelLatestVersionNumber(modelID string) (uint, error) {
	latestVersionInfo, err := b.retrieveModelNthToLastVersionInfo(modelID, 0)
	if err != nil {
		return 0, err
	}
	return latestVersionInfo.VersionNumber, nil
}

// HasModel checks if a model exists
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
	_, err := b.retrieveModelNthToLastVersionInfo(modelID, 0)
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
	if limit > 0 {
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
func (b *fsBackend) ListModels(offset int, limit int) ([]backend.ModelInfo, error) {
	modelEntries, err := filteredReadDir(b.rootDirname, offset, limit, func(entry fs.DirEntry) bool {
		return entry.IsDir() && modelDirnameRegexp.MatchString(entry.Name())
	})
	if err != nil {
		return []backend.ModelInfo{}, fmt.Errorf("unable to list models: %w", err)
	}

	models := []backend.ModelInfo{}
	for _, entry := range modelEntries {
		modelID := entry.Name()
		modelInfoFilename := b.buildModelInfoFilename(backend.ModelInfo{ModelID: modelID})

		_, err := os.Stat(modelInfoFilename)
		if err != nil {
			return []backend.ModelInfo{}, fmt.Errorf("unable to read model info from %q: %w", modelInfoFilename, err)
		}

		modelInfo, err := loadModelInfoFile(modelInfoFilename)
		if err != nil {
			return []backend.ModelInfo{}, err
		}

		models = append(models, modelInfo)
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

func (b *fsBackend) buildModelInfoFilename(modelInfo backend.ModelInfo) string {
	modelInfoFilenameBuffer := new(bytes.Buffer)
	err := modelInfoFilenameTemplate.Execute(modelInfoFilenameBuffer, modelInfo)
	if err != nil {
		panic(err)
	}
	return path.Join(b.rootDirname, modelInfo.ModelID, modelInfoFilenameBuffer.String())
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
func (b *fsBackend) CreateOrUpdateModelVersion(
	modelID string,
	versionArgs backend.VersionArgs,
) (backend.VersionInfo, error) {
	var versionInfo backend.VersionInfo
	if versionArgs.VersionNumber == 0 {
		// Create a new version after the last one
		latestVersionInfo, err := b.retrieveModelNthToLastVersionInfo(modelID, 0)
		if err != nil {
			return backend.VersionInfo{}, err
		}
		versionInfo = backend.VersionInfo{
			ModelID:           modelID,
			VersionNumber:     latestVersionInfo.VersionNumber + 1,
			CreationTimestamp: versionArgs.CreationTimestamp,
			Archived:          versionArgs.Archived,
			DataHash:          versionArgs.DataHash,
			DataSize:          len(versionArgs.Data),
			UserData:          versionArgs.UserData,
		}
	} else {
		// Maybe there is an existing version
		existingVersionInfo, err := b.RetrieveModelVersionInfo(modelID, int(versionArgs.VersionNumber))
		if err != nil {
			if _, ok := err.(*backend.UnknownModelVersionError); !ok {
				return backend.VersionInfo{}, err
			}
			// No version, create a new one
			versionInfo = backend.VersionInfo{
				ModelID:           modelID,
				VersionNumber:     versionArgs.VersionNumber,
				CreationTimestamp: versionArgs.CreationTimestamp,
				Archived:          versionArgs.Archived,
				DataHash:          versionArgs.DataHash,
				DataSize:          len(versionArgs.Data),
				UserData:          versionArgs.UserData,
			}
		} else {
			// Update an existing version
			versionInfo = existingVersionInfo
			versionInfo.Archived = versionArgs.Archived
			versionInfo.DataHash = versionArgs.DataHash
			versionInfo.DataSize = len(versionArgs.Data)
			versionInfo.UserData = versionArgs.UserData
		}
	}

	versionInfoFilename := b.buildVersionInfoFilename(versionInfo)

	err := saveVersionInfoFile(versionInfoFilename, versionInfo)
	if err != nil {
		return backend.VersionInfo{}, err
	}

	versionDataFilename := b.buildVersionDataFilename(versionInfo)

	err = lockedfile.Write(versionDataFilename, bytes.NewReader(versionArgs.Data), 0640)
	if err != nil {
		os.Remove(versionInfoFilename)
		return backend.VersionInfo{}, fmt.Errorf("unable to create a version for model %q: %w", modelID, err)
	}

	return versionInfo, nil
}

func (b *fsBackend) RetrieveModelLastVersionInfo(modelID string) (backend.VersionInfo, error) {
	return b.retrieveModelNthToLastVersionInfo(modelID, 0)
}

// RetrieveModelVersionInfo retrieves a given model version info
func (b *fsBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	if versionNumber == 0 {
		return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
	}
	if versionNumber < 0 {
		// Retrieve the nth to last version
		versionInfo, err := b.retrieveModelNthToLastVersionInfo(modelID, uint(-versionNumber-1))
		if err != nil {
			return backend.VersionInfo{}, err
		}
		if versionInfo.VersionNumber == 0 {
			return backend.VersionInfo{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return versionInfo, nil
	}
	// Retrieve a specific version
	versionInfoFilename := b.buildVersionInfoFilename(
		backend.VersionInfo{ModelID: modelID,
			VersionNumber: uint(versionNumber)},
	)
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
	var versionInfo backend.VersionInfo
	if versionNumber == 0 {
		return []byte{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
	}
	if versionNumber < 0 {
		// Retrieve the nth to last version
		var err error
		versionInfo, err = b.retrieveModelNthToLastVersionInfo(modelID, uint(-versionNumber-1))
		if err != nil {
			return []byte{}, err
		}
		if versionInfo.VersionNumber == 0 {
			return []byte{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
	} else {
		versionInfo = backend.VersionInfo{
			ModelID:       modelID,
			VersionNumber: uint(versionNumber),
		}
	}

	// Retrieve a specific version
	versionDataFilename := b.buildVersionDataFilename(versionInfo)
	versionData, err := os.ReadFile(versionDataFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return []byte{}, fmt.Errorf(
			`unable to read data for model %q version "%d": %w`,
			versionInfo.ModelID, versionInfo.VersionNumber, err,
		)
	}
	return versionData, nil
}

// DeleteModelVersion deletes a given model version
func (b *fsBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	var versionInfo backend.VersionInfo
	if versionNumber == 0 {
		return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
	}
	if versionNumber < 0 {
		// Retrieve the nth to last version
		var err error
		latestVersionInfo, err := b.retrieveModelNthToLastVersionInfo(modelID, uint(-versionNumber-1))
		if err != nil {
			return err
		}
		if latestVersionInfo.VersionNumber == 0 {
			return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		versionInfo = latestVersionInfo
	} else {
		versionInfo = backend.VersionInfo{
			ModelID:       modelID,
			VersionNumber: uint(versionNumber),
		}
	}
	versionInfoFilename := b.buildVersionInfoFilename(versionInfo)
	err := os.Remove(versionInfoFilename)
	if err != nil {
		if os.IsNotExist(err) {
			return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return fmt.Errorf(`unable to delete model %q version "%d" info: %w`, modelID, versionNumber, err)
	}
	versionDataFilename := b.buildVersionDataFilename(versionInfo)
	err = os.Remove(versionDataFilename)
	if err != nil {
		return fmt.Errorf(`unable to delete model %q version "%d" data: %w`, modelID, versionNumber, err)
	}
	return nil
}

func (b *fsBackend) ListModelVersionInfos(
	modelID string,
	initialVersionNumber uint,
	limit int,
) ([]backend.VersionInfo, error) {
	modelDirname := path.Join(b.rootDirname, modelID)
	modelVersionEntries, err := filteredReadDir(modelDirname, 0, limit, func(entry fs.DirEntry) bool {
		if entry.IsDir() {
			return false
		}
		submatches := versionInfoFilenameRegexp.FindStringSubmatch(entry.Name())
		if submatches == nil {
			return false
		}
		// Parsing the version number, we ignore the error as the regex matching should be enough guarantees
		versionNumber, _ := strconv.ParseUint(submatches[2], 10, 0)
		return versionNumber >= uint64(initialVersionNumber)
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
