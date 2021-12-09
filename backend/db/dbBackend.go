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

package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/cogment/cogment-model-registry/backend"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type dbModel struct {
	ModelID  string `gorm:"primarykey"`
	UserData []byte

	CreationTimestamp time.Time   `gorm:"autoCreateTime"`
	Versions          []dbVersion `gorm:"foreignKey:ModelID;reference:ModelID"`
}

type dbVersion struct {
	ModelID           string `gorm:"primarykey"`
	VersionNumber     int    `gorm:"primarykey"`
	CreationTimestamp time.Time
	Archived          bool
	DataHash          string
	DataSize          int
	Data              []byte
	UserData          []byte
}

type dbVersionInfo struct {
	ModelID           string    `gorm:"primarykey"`
	VersionNumber     int       `gorm:"primarykey"`
	CreationTimestamp time.Time `gorm:"autoCreateTime"`
	Archived          bool
	DataHash          string
	DataSize          int
	UserData          []byte
}

type dbBackend struct {
	db *gorm.DB
}

func backendVersionInfoFromDB(versionInfo dbVersionInfo) (backend.VersionInfo, error) {

	versionUserData := &map[string]string{}
	err := json.Unmarshal(versionInfo.UserData, versionUserData)
	if err != nil {
		return backend.VersionInfo{}, err
	}

	return backend.VersionInfo{
		ModelID:           versionInfo.ModelID,
		CreationTimestamp: versionInfo.CreationTimestamp,
		VersionNumber:     versionInfo.VersionNumber,
		Archived:          versionInfo.Archived,
		DataHash:          versionInfo.DataHash,
		DataSize:          versionInfo.DataSize,
		UserData:          *versionUserData,
	}, nil
}

func backendModelInfoFromDB(ModelInfo dbModel) (backend.ModelInfo, error) {
	modelUserData := &map[string]string{}
	err := json.Unmarshal(ModelInfo.UserData, modelUserData)
	if err != nil {
		return backend.ModelInfo{}, err
	}

	return backend.ModelInfo{
		ModelID:  ModelInfo.ModelID,
		UserData: *modelUserData,
	}, nil
}

// CreateBackend creates a new backend with memory storage in a SQLite Database
func CreateBackend() (backend.Backend, error) {
	newLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
	})

	b := &dbBackend{}
	dbDir, err := ioutil.TempDir("", "model-registry")
	if err != nil {
		return nil, fmt.Errorf("Error while creating a temp directory for the database: %w", err)
	}

	// Setting cache=shared following the guidelines from https://github.com/mattn/go-sqlite3#faq
	db, err := gorm.Open(sqlite.Open(path.Join(dbDir, "sqlite.db")+"?cache=shared"), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, fmt.Errorf("Error while connecting to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("Unexpected error while accessing the low level SQL DB driver: %w", err)
	}
	sqlDB.SetMaxOpenConns(1) // Following the guidelines from https://github.com/mattn/go-sqlite3#faq

	err = db.AutoMigrate(&dbModel{}, &dbVersion{})
	if err != nil {
		return nil, fmt.Errorf("Error during database migration: %w", err)
	}

	b.db = db
	return b, nil
}

// Destroy terminates the underlying storage
func (b *dbBackend) Destroy() {
	// Nothing
}

func (b *dbBackend) retrieveLatestVersionNumber(db *gorm.DB, modelID string) (int, error) {
	latestVersionInfo := dbVersion{ModelID: modelID}
	if err := db.Order("version_number desc").First(&latestVersionInfo).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			// No version yet
			return 0, nil
		} else {
			return 0, err
		}
	}
	return latestVersionInfo.VersionNumber, nil
}

// CreateModel creates a model with a given unique id in the backend
func (b *dbBackend) CreateOrUpdateModel(modelInfo backend.ModelInfo) (backend.ModelInfo, error) {

	serializedUserData, err := json.Marshal(modelInfo.UserData)

	if err != nil {
		return backend.ModelInfo{}, err
	}

	model := dbModel{ModelID: modelInfo.ModelID, UserData: serializedUserData}
	tx := b.db.Begin()

	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "model_id"}},             // key column
		DoUpdates: clause.AssignmentColumns([]string{"user_data"}), // column needed to be updated
	}).Create(&model).Error; err != nil {
		tx.Rollback()
		return backend.ModelInfo{}, err
	}

	latestVersionNumber, err := b.retrieveLatestVersionNumber(tx, modelInfo.ModelID)
	if err != nil {
		// Unexpected error
		tx.Rollback()
		return backend.ModelInfo{}, err
	}

	modelInfo.LatestVersionNumber = latestVersionNumber

	tx.Commit()
	return modelInfo, nil
}

func retrieveModel(db *gorm.DB, modelID string, dest interface{}) error {
	if err := db.First(dest, "model_id=?", modelID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return &backend.UnknownModelError{ModelID: modelID}
		}
		return err
	}
	return nil
}

// RetrieveModelVersionInfo retrieves a given model version info
func (b *dbBackend) RetrieveModelInfo(modelID string) (backend.ModelInfo, error) {
	tx := b.db.Begin()

	modelInfo := dbModel{}
	if err := retrieveModel(tx, modelID, &modelInfo); err != nil {
		tx.Rollback()
		return backend.ModelInfo{}, err
	}

	modelInfoOut, err := backendModelInfoFromDB(modelInfo)
	if err != nil {
		tx.Rollback()
		return backend.ModelInfo{}, err
	}

	latestVersionNumber, err := b.retrieveLatestVersionNumber(tx, modelInfo.ModelID)
	if err != nil {
		// Unexpected error
		tx.Rollback()
		return backend.ModelInfo{}, err
	}

	modelInfoOut.LatestVersionNumber = latestVersionNumber

	tx.Commit()
	return modelInfoOut, nil
}

// HasModel check if a model exists
func (b *dbBackend) HasModel(modelID string) (bool, error) {
	modelInfo := dbModel{}
	if err := retrieveModel(b.db, modelID, &modelInfo); err != nil {
		if _, ok := err.(*backend.UnknownModelError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteModel deletes a model with a given id from the storage
func (b *dbBackend) DeleteModel(modelID string) error {
	tx := b.db.Begin()

	model := dbModel{}

	if err := tx.First(&model, "model_id=?", modelID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			tx.Rollback()
			return &backend.UnknownModelError{ModelID: modelID}
		}
		tx.Rollback()
		return err
	}

	if err := tx.Select("Versions").Delete(&model).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// ListModels list models ordered by id from the given offset index, it returns at most the given limit VersionNumber of models
func (b *dbBackend) ListModels(offset int, limit int) ([]string, error) {
	models := []dbModel{}
	if limit <= 0 {
		limit = -1
	}
	if err := b.db.Order("model_id").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return []string{}, err
	}
	modelIDs := []string{}
	for _, model := range models {
		modelIDs = append(modelIDs, model.ModelID)
	}
	return modelIDs, nil
}

// CreateOrUpdateModelVersion creates and store a new version for a model and returns its info, including the version VersionNumber
func (b *dbBackend) CreateOrUpdateModelVersion(modelID string, versionArgs backend.VersionArgs) (backend.VersionInfo, error) {
	tx := b.db.Begin()

	if err := tx.First(&dbModel{}, "model_id=?", modelID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			tx.Rollback()
			return backend.VersionInfo{}, &backend.UnknownModelError{ModelID: modelID}
		}
		tx.Rollback()
		return backend.VersionInfo{}, err
	}

	serializedUserData, err := json.Marshal(versionArgs.UserData)

	if err != nil {
		return backend.VersionInfo{}, err
	}

	version := dbVersion{
		ModelID:           modelID,
		VersionNumber:     versionArgs.VersionNumber,
		CreationTimestamp: versionArgs.CreationTimestamp,
		Archived:          versionArgs.Archived,
		Data:              versionArgs.Data,
		DataHash:          versionArgs.DataHash,
		DataSize:          len(versionArgs.Data),
		UserData:          serializedUserData,
	}

	if version.VersionNumber <= 0 {
		latestVersionNumber, err := b.retrieveLatestVersionNumber(tx, modelID)
		if err != nil {
			// Unexpected error
			tx.Rollback()
			return backend.VersionInfo{}, err
		}
		version.VersionNumber = latestVersionNumber + 1
	}

	if err := tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&version).Error; err != nil {
		tx.Rollback()
		return backend.VersionInfo{}, err
	}

	tx.Commit()

	versionUserData := &map[string]string{}
	err = json.Unmarshal(version.UserData, versionUserData)
	if err != nil {
		return backend.VersionInfo{}, err
	}

	return backend.VersionInfo{
		ModelID:           version.ModelID,
		CreationTimestamp: version.CreationTimestamp,
		Archived:          version.Archived,
		VersionNumber:     version.VersionNumber,
		DataHash:          version.DataHash,
		DataSize:          version.DataSize,
		UserData:          *versionUserData,
	}, nil
}

func retrieveModelVersion(db *gorm.DB, modelID string, versionNumber int, dest interface{}) error {
	if versionNumber <= 0 {
		// Retrieving the latest version
		if err := db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID}).Order("version_number desc").First(dest).Error; err != nil {
			if errors.Is(gorm.ErrRecordNotFound, err) {
				return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: -1}
			}
			return err
		}
		return nil
	}
	// Retrieving a specific version
	if err := db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID, VersionNumber: versionNumber}).First(dest).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: versionNumber}
		}
		return err
	}
	return nil
}

// RetrieveModelVersionData retrieves a given model version data
func (b *dbBackend) RetrieveModelVersionData(modelID string, versionNumber int) ([]byte, error) {
	version := dbVersion{}
	if err := retrieveModelVersion(b.db, modelID, versionNumber, &version); err != nil {
		return []byte{}, err
	}
	return version.Data, nil
}

// RetrieveModelVersionInfo retrieves a given model version info
func (b *dbBackend) RetrieveModelVersionInfo(modelID string, versionNumber int) (backend.VersionInfo, error) {
	versionInfo := dbVersionInfo{}
	if err := retrieveModelVersion(b.db, modelID, versionNumber, &versionInfo); err != nil {
		return backend.VersionInfo{}, err
	}

	versionInfoOut, err := backendVersionInfoFromDB(versionInfo)
	if err != nil {
		return backend.VersionInfo{}, err
	}

	return versionInfoOut, nil
}

// DeleteModelVersion deletes a given model version
func (b *dbBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	tx := b.db.Begin()

	version := dbVersion{}
	if err := retrieveModelVersion(tx, modelID, versionNumber, &version); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("model_id=?", modelID).Where("version_number=?", versionNumber).Delete(&dbVersion{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// ListModelVersionInfos list the versions info of a model from the latest to the earliest from the given offset index, it returns at most the given limit VersionNumber of versions
func (b *dbBackend) ListModelVersionInfos(modelID string, offset int, limit int) ([]backend.VersionInfo, error) {
	versions := []dbVersionInfo{}
	if err := b.db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID}).Limit(limit).Offset(offset).Find(&versions).Error; err != nil {
		return []backend.VersionInfo{}, err
	}

	outVersions := []backend.VersionInfo{}
	for _, version := range versions {
		outVersion, err := backendVersionInfoFromDB(version)
		if err != nil {
			return []backend.VersionInfo{}, err
		}
		outVersions = append(outVersions, outVersion)
	}

	return outVersions, nil
}
