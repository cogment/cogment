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
	ModelID   string `gorm:"primarykey"`
	CreatedAt time.Time
	Versions  []dbVersion `gorm:"foreignKey:ModelID;reference:ModelID"`
}

type dbVersion struct {
	CreatedAt time.Time
	ModelID   string `gorm:"primarykey"`
	Number    int    `gorm:"primarykey"`
	Archive   bool
	Hash      string
	Data      []byte
}

type dbBackend struct {
	db *gorm.DB
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

	db, err := gorm.Open(sqlite.Open(path.Join(dbDir, "sqlite.db")), &gorm.Config{Logger: newLogger})
	if err != nil {
		return nil, fmt.Errorf("Error while connecting to database: %w", err)
	}

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

// CreateModel creates a model with a given unique id in the backend
func (b *dbBackend) CreateModel(modelID string) error {
	model := dbModel{ModelID: modelID}
	tx := b.db.Begin()

	// Check if there's already a model with the same id
	var modelCount int64
	if err := tx.Model(&model).Where(&model).Count(&modelCount).Error; err != nil {
		tx.Rollback()
		return err
	}

	if modelCount > 0 {
		tx.Rollback()
		return &backend.ModelAlreadyExistsError{ModelID: modelID}
	}

	// Create the model
	if err := tx.Create(&model).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// HasModel check if a model exists
func (b *dbBackend) HasModel(modelID string) (bool, error) {
	model := dbModel{}
	if err := b.db.First(&model, "model_id=?", modelID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
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

// ListModels list models ordered by id from the given offset index, it returns at most the given limit number of models
func (b *dbBackend) ListModels(offset int, limit int) ([]string, error) {
	models := []dbModel{}
	if err := b.db.Order("model_id").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return []string{}, err
	}
	modelIDs := []string{}
	for _, model := range models {
		modelIDs = append(modelIDs, model.ModelID)
	}
	return modelIDs, nil
}

// CreateOrUpdateModelVersion creates and store a new version for a model and returns its info, including the version number
func (b *dbBackend) CreateOrUpdateModelVersion(modelID string, versionNumber int, data []byte, archive bool) (backend.VersionInfo, error) {
	tx := b.db.Begin()

	if err := tx.First(&dbModel{}, "model_id=?", modelID).Error; err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			tx.Rollback()
			return backend.VersionInfo{}, &backend.UnknownModelError{ModelID: modelID}
		}
		tx.Rollback()
		return backend.VersionInfo{}, err
	}

	version := dbVersion{
		ModelID: modelID,
		Number:  versionNumber,
		Archive: archive,
		Data:    data,
		Hash:    backend.ComputeHash(data),
	}

	if version.Number <= 0 {
		// If versionNumber is 0 or less, we create a new version after the last one
		latestVersionInfo := dbVersion{ModelID: modelID}
		if err := tx.Order("number desc").First(&latestVersionInfo).Error; err != nil {
			if errors.Is(gorm.ErrRecordNotFound, err) {
				// No version yet
				version.Number = 1
			} else {
				// Unexpected error
				tx.Rollback()
				return backend.VersionInfo{}, err
			}
		} else {
			version.Number = latestVersionInfo.Number + 1
		}
	}

	if err := tx.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&version).Error; err != nil {
		tx.Rollback()
		return backend.VersionInfo{}, err
	}

	tx.Commit()
	return backend.VersionInfo{
		ModelID:   version.ModelID,
		CreatedAt: version.CreatedAt,
		Archive:   version.Archive,
		Number:    version.Number,
		Hash:      version.Hash,
	}, nil
}

func retrieveModelVersion(db *gorm.DB, modelID string, versionNumber int, dest interface{}) error {
	if versionNumber <= 0 {
		// Retrieving the latest version
		if err := db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID}).Order("number desc").First(dest).Error; err != nil {
			if errors.Is(gorm.ErrRecordNotFound, err) {
				return &backend.UnknownModelVersionError{ModelID: modelID, VersionNumber: -1}
			}
			return err
		}
		return nil
	}
	// Retrieving a specific version
	if err := db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID, Number: versionNumber}).First(dest).Error; err != nil {
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
	versionInfo := backend.VersionInfo{}
	if err := retrieveModelVersion(b.db, modelID, versionNumber, &versionInfo); err != nil {
		return backend.VersionInfo{}, err
	}
	return versionInfo, nil
}

// DeleteModelVersion deletes a given model version
func (b *dbBackend) DeleteModelVersion(modelID string, versionNumber int) error {
	tx := b.db.Begin()

	version := dbVersion{}
	if err := retrieveModelVersion(tx, modelID, versionNumber, &version); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("model_id=?", modelID).Where("number=?", versionNumber).Delete(&dbVersion{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// ListModelVersionInfos list the versions info of a model from the latest to the earliest from the given offset index, it returns at most the given limit number of versions
func (b *dbBackend) ListModelVersionInfos(modelID string, offset int, limit int) ([]backend.VersionInfo, error) {
	versions := []backend.VersionInfo{}
	if err := b.db.Model(&dbVersion{}).Where(&dbVersion{ModelID: modelID}).Limit(limit).Offset(offset).Find(&versions).Error; err != nil {
		return []backend.VersionInfo{}, err
	}
	return versions, nil
}
