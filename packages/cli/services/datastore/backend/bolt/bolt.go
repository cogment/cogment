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

package bolt

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sort"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	grpcapi "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/services/datastore/backend"
)

type boltBackend struct {
	db                    *bolt.DB
	filePath              string
	observeDbPollingDelay time.Duration // The maximum duration between two polling of the db during an 'observe' request
}

// metadata includes all the data used to filter trials and retrieve the trial info datastructure
type metadata struct {
	UserID     string
	TrialIdx   uint64
	Properties map[string]string
}

// Bucket structure is
//	trials	> {trial_id}			> samples			> {tick_id}	> {grpcapi.StoredTrialSample}
//                                  > sampleCount       > {int}
//									> params			> {grpcapi.TrialParams}
//									> metadata		    > {boltBackend.metadata}
//	trial_indices	>	trial_idx	>	{trial_idx}	>	{trial_id}
//  schema_history  > {schema_version} > emptyarray

// {schema_version}: function_to_run_when_migrating
var migrationCatalog = map[string]func(*boltBackend) error{
	"2.20.0": computeStoredSampleCount,
}

var trialsBucketName = []byte("trials")

func getTrialsBucket(tx *bolt.Tx) *bolt.Bucket {
	trialsBucket := tx.Bucket(trialsBucketName)
	if trialsBucket == nil {
		log.Fatal("trials bucket doesn't exist")
	}
	return trialsBucket
}

// NOTE: bbolt key-values are stored as []byte
var samplesBucketName = []byte("samples")

var paramsKey = []byte("params")

var metadataKey = []byte("metadata")

var sampleCountKey = []byte("sample_count")

var indicesBucketName = []byte("trial_indices")

var trialsIdxBucketName = []byte("trial_idx")

var schemaHistoryBucketName = []byte("schema_history")

func getSchemaHistoryBucket(tx *bolt.Tx) *bolt.Bucket {
	bucket := tx.Bucket(schemaHistoryBucketName)
	if bucket == nil {
		log.Fatal("schema history bucket doesn't exist")
	}
	return bucket
}

func getTrialsIdxBucket(tx *bolt.Tx) *bolt.Bucket {
	indicesBucket := tx.Bucket(indicesBucketName)
	if indicesBucket == nil {
		log.Fatal("indices bucket doesn't exist")
	}
	trialsIdxBucket := indicesBucket.Bucket(trialsIdxBucketName)
	if trialsIdxBucket == nil {
		log.Fatalf("trials idx bucket doesn't exist")
	}
	return trialsIdxBucket
}

func serializeNumID(id uint64) []byte {
	// Format using a hex representation of a fixed length of 16 characters padded with 0
	return []byte(fmt.Sprintf("%016x", id))
}

func deserializeNumIDAsInt(value []byte) (int, error) {
	number, err := strconv.ParseInt(string(value), 16, 32)
	if err != nil {
		return 0, backend.NewUnexpectedError("unable to deserialize number id as an int (%w)", err)
	}
	return int(number), nil
}

func serializeTrialID(trialID string) []byte {
	return []byte(trialID)
}

func deserializeTrialID(value []byte) string {
	return string(value)
}

func serializeTrialParams(params *grpcapi.TrialParams) ([]byte, error) {
	v, err := proto.Marshal(params)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to serialize trial params (%w)", err)
	}
	return v, nil
}

func deserializeTrialParams(v []byte) (*grpcapi.TrialParams, error) {
	params := &grpcapi.TrialParams{}
	err := proto.Unmarshal(v, params)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to deserialize trial params (%w)", err)
	}
	return params, nil
}

func serializeSample(sample *grpcapi.StoredTrialSample) ([]byte, error) {
	v, err := proto.Marshal(sample)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to serialize sample (%w)", err)
	}
	return v, nil
}

func deserializeSample(v []byte) (*grpcapi.StoredTrialSample, error) {
	sample := &grpcapi.StoredTrialSample{}
	err := proto.Unmarshal(v, sample)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to deserialize sample (%w)", err)
	}
	return sample, nil
}

func serializeTrialMetadata(metadata *metadata) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(*metadata)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to serialize trial metadata (%w)", err)
	}
	return buf.Bytes(), nil
}

func deserializeTrialMetadata(v []byte) (*metadata, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(v))
	metadata := &metadata{}
	err := dec.Decode(metadata)
	if err != nil {
		return nil, backend.NewUnexpectedError("unable to deserialize trial metadata (%w)", err)
	}
	return metadata, nil
}

func serializeMigrationVersion(version string) []byte {
	return []byte(version)
}

func deserializeMigrationVersion(value []byte) string {
	return string(value)
}

func (b *boltBackend) persistSucessfulMigration(version string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := getSchemaHistoryBucket(tx)
		err := bucket.Put(serializeMigrationVersion(version), []byte{})
		if err != nil {
			return backend.NewUnexpectedError("unable to persist successful migration (%v) (%w)", version, err)
		}
		return nil
	})
	return err
}

func (b *boltBackend) readSuccessfulMigrations() ([]string, error) {
	var acc = []string{}
	err := b.db.View(func(tx *bolt.Tx) error {
		schemaHistory := getSchemaHistoryBucket(tx)
		return schemaHistory.ForEach(func(k, v []byte) error {
			acc = append(acc, deserializeMigrationVersion(k))
			return nil
		})
	})
	return acc, err
}

func (b *boltBackend) runMigrations() error {

	appliedMigrations, err := b.readSuccessfulMigrations()
	if err != nil {
		return err
	}
	log.Infof("Migrations already applied (%v)", appliedMigrations)

	pendingMigrations := make([]string, 0, len(migrationCatalog))
	for version := range migrationCatalog {
		found := false
		for _, migrationVersion := range appliedMigrations {
			log.Debugf("Comparing migrations (%v) (%v)", version, migrationVersion)
			if version == migrationVersion {
				found = true
				break
			}
		}
		if !found {
			pendingMigrations = append(pendingMigrations, version)
		}
	}

	if len(pendingMigrations) == 0 {
		log.Info("No pending migration")
		return nil
	}

	sort.Strings(pendingMigrations)

	for _, version := range pendingMigrations {
		log.Infof("Migrating to version (%v)", version)

		err := migrationCatalog[version](b)
		if err != nil {
			return err
		}
		err = b.persistSucessfulMigration(version)
		if err != nil {
			return err
		}
		log.Infof("Migrated to version (%v)", version)
	}

	return nil
}

// CreateBoltBackend creates a Backend that will store samples in a blot-managed file
func CreateBoltBackend(filePath string) (backend.Backend, error) {
	db, err := bolt.Open(filePath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		// Opening of the file failed
		return nil, err
	}
	// Create the root buckets
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(trialsBucketName)
		if err != nil {
			return backend.NewUnexpectedError("unable to create the trials bucket (%w)", err)
		}
		indicesBucket, err := tx.CreateBucketIfNotExists(indicesBucketName)
		if err != nil {
			return backend.NewUnexpectedError("unable to create the trial indices bucket (%w)", err)
		}
		_, err = indicesBucket.CreateBucketIfNotExists(trialsIdxBucketName)
		if err != nil {
			return backend.NewUnexpectedError("unable to create the trial idx bucket (%w)", err)
		}
		_, err = tx.CreateBucketIfNotExists(schemaHistoryBucketName)
		if err != nil {
			return backend.NewUnexpectedError("unable to create the schema history bucket (%w)", err)
		}
		return nil
	})
	if err != nil {
		// Creation of the root buckets failed
		return nil, err
	}

	bolt := boltBackend{
		db:                    db,
		filePath:              filePath,
		observeDbPollingDelay: 100 * time.Millisecond,
	}
	err = bolt.runMigrations()
	if err != nil {
		return nil, err
	}

	return &bolt, nil
}

func (b *boltBackend) Destroy() {
	b.db.Close()
	b.db = nil
}

func (b *boltBackend) CreateOrUpdateTrials(_ context.Context, paramsList []*backend.TrialParams) error {
	err := b.db.Batch(func(tx *bolt.Tx) error {
		// Function must be idempotent as it might be called multiple times
		trialsBucket := getTrialsBucket(tx)
		trialsIdxBucket := getTrialsIdxBucket(tx)
		for _, params := range paramsList {
			trialKey := serializeTrialID(params.TrialID)
			var trialIdx uint64
			trialBucket := trialsBucket.Bucket(trialKey)

			if trialBucket == nil {
				// This is a new trial, inserting its idx
				var err error
				trialBucket, err = trialsBucket.CreateBucket(trialKey)
				if err != nil {
					return backend.NewUnexpectedError("unable to add trial %q bucket (%w)", params.TrialID, err)
				}

				// Initialize new trial sample count to zero
				err = trialBucket.Put(sampleCountKey, serializeNumID(uint64(0)))
				if err != nil {
					return backend.NewUnexpectedError(
						"unable to put sample count [%d] for trial [%q]: %w",
						uint64(0), params.TrialID, err,
					)
				}

				// Because we use `NextSequence` here the trialIdx starts at 1
				// Changing it would break backward compatibility with previous storage though
				trialIdx, _ = trialsIdxBucket.NextSequence()
				trialIdxKey := serializeNumID(trialIdx)
				err = trialsIdxBucket.Put(trialIdxKey, trialKey)
				if err != nil {
					return backend.NewUnexpectedError("unable to add trial %q insertion index (%w)", params.TrialID, err)
				}
			} else {
				// This is an existing trial, retrieving its idx
				metadataV := trialBucket.Get(metadataKey)
				if metadataV == nil {
					return backend.NewUnexpectedError("no metadata for trial %q", params.TrialID)
				}
				metadata, err := deserializeTrialMetadata(metadataV)
				if err != nil {
					return err
				}
				trialIdx = metadata.TrialIdx
			}

			// Create sample bucket if it doesn't exist
			_, err := trialBucket.CreateBucketIfNotExists(samplesBucketName)
			if err != nil {
				return backend.NewUnexpectedError("unable to add trial %q sample bucket (%w)", params.TrialID, err)
			}

			// Insert / Update metadata
			metadataV, err := serializeTrialMetadata(&metadata{
				UserID:     params.UserID,
				TrialIdx:   trialIdx,
				Properties: params.Params.Properties,
			})
			if err != nil {
				return err
			}

			err = trialBucket.Put(metadataKey, metadataV)
			if err != nil {
				return backend.NewUnexpectedError("unable to add trial %q metadata (%w)", params.TrialID, err)
			}

			// Insert / Update trial params
			paramsV, err := serializeTrialParams(params.Params)
			if err != nil {
				return err
			}

			err = trialBucket.Put(paramsKey, paramsV)
			if err != nil {
				return backend.NewUnexpectedError("unable to add trial %q params (%w)", params.TrialID, err)
			}
		}
		return nil
	})

	if err != nil {
		// Error during the insertion
		return err
	}

	return nil
}

func computeStoredSampleCount(b *boltBackend) error {
	log.Debug("Migrating bolt backend to precomputed sample count ...")
	log.Debug("Listing trials ...")
	trialIDKeys := [][]byte{}
	err := b.db.View(func(tx *bolt.Tx) error {
		trialsIdxBucket := getTrialsIdxBucket(tx)
		err := trialsIdxBucket.ForEach(func(k, v []byte) error {
			trialIDKeys = append(trialIDKeys, v)
			return nil
		})
		return err
	})
	if err != nil {
		return err
	}
	trialCount := len(trialIDKeys)
	log.Debugf("Listing trials - Done %v", trialCount)

	for i, key := range trialIDKeys {
		err := b.db.Update(func(tx *bolt.Tx) error {
			trialBucket := getTrialsBucket(tx).Bucket(key)
			log.Debugf("Computing sampleCount for trial [%q] [%v/%v] ...", key, i, trialCount)
			sampleCount := getSampleCount(trialBucket, deserializeTrialID(key))
			log.Debugf("Computing sampleCount for trial [%q] [%v/%v] - Done counting [%v] samples",
				key, i, trialCount, sampleCount)
			err := trialBucket.Put(sampleCountKey, serializeNumID(uint64(sampleCount)))
			return err
		})
		if err != nil {
			return err
		}
	}
	log.Debug("Migrating bolt backend to precomputed sample count - Done")

	return nil
}

func getSampleCount(trialBucket *bolt.Bucket, trialID string) int {

	byteValue := trialBucket.Get(sampleCountKey)
	if byteValue == nil {
		log.Warnf("No precomputed sample count for trialID [%q]. Computing from scratch", trialID)
		sampleCount := trialBucket.Bucket(samplesBucketName).Stats().KeyN
		return sampleCount
	}

	count, err := deserializeNumIDAsInt(byteValue)
	if err != nil {
		log.Warnf("Could not deserialize sampleCount for trialID [%q]. Computing from scratch", trialID)
		sampleCount := trialBucket.Bucket(samplesBucketName).Stats().KeyN
		return sampleCount
	}

	return count
}

func (b *boltBackend) RetrieveTrials(
	_ context.Context,
	filter backend.TrialFilter,
	fromTrialIdx int,
	count int,
) (backend.TrialsInfoResult, error) {
	trialInfos := []*backend.TrialInfo{}
	nextTrialIdx := 0
	err := b.db.View(func(tx *bolt.Tx) error {
		trialsBucket := getTrialsBucket(tx)
		trialsIdxBucket := getTrialsIdxBucket(tx)

		var trialIdxKey []byte
		var trialIDKey []byte
		c := trialsIdxBucket.Cursor()
		if fromTrialIdx <= 0 {
			trialIdxKey, trialIDKey = c.First()
		} else {
			// Adding +1 because the stored trialIdx offset
			trialIdxKey, trialIDKey = c.Seek(serializeNumID(uint64(fromTrialIdx + 1)))
		}
		for ; trialIdxKey != nil; trialIdxKey, trialIDKey = c.Next() {
			if count > 0 && len(trialInfos) >= count {
				// We've retrieved enough trialInfos
				break
			}
			trialID := deserializeTrialID(trialIDKey)
			if filter.IDFilter.Selects(trialID) {
				trialBucket := trialsBucket.Bucket(trialIDKey)
				if trialBucket == nil {
					return backend.NewUnexpectedError("no bucket for trial %q", trialID)
				}

				// Retrieve the trial metadata
				trialMetadataV := trialBucket.Get(metadataKey)
				if trialMetadataV == nil {
					return backend.NewUnexpectedError("no metadata for trial %q", trialID)
				}

				metadata, err := deserializeTrialMetadata(trialMetadataV)
				if err != nil {
					return err
				}

				if !filter.PropertiesFilter.Selects(metadata.Properties) {
					continue
				}

				// Retrieve the number of samples
				samplesBucket := trialBucket.Bucket(samplesBucketName)
				if samplesBucket == nil {
					return backend.NewUnexpectedError("no sample bucket for trial %q", trialID)
				}
				samplesCount := getSampleCount(trialBucket, trialID)

				// Retrieve the last samples
				state := grpcapi.TrialState_UNKNOWN
				if samplesCount > 0 {
					_, v := samplesBucket.Cursor().Last()
					lastSample := &grpcapi.StoredTrialSample{}
					err := proto.Unmarshal(v, lastSample)
					if err != nil {
						return backend.NewUnexpectedError("unable to deserialize the last stored sample of trial %q", trialID)
					}
					state = lastSample.State
				}
				trialInfos = append(trialInfos, &backend.TrialInfo{
					TrialID:            trialID,
					UserID:             metadata.UserID,
					State:              state,
					SamplesCount:       samplesCount,
					StoredSamplesCount: samplesCount,
				})
			}
		}

		if trialIdxKey != nil {
			var err error
			nextTrialIdx, err = deserializeNumIDAsInt(trialIdxKey)
			if err != nil {
				return err
			}
			// Dealing with internal index being offseted
			nextTrialIdx--
		} else {
			// TODO Handle overflow ?
			nextTrialIdx = int(trialsIdxBucket.Sequence())
		}

		return nil
	})

	if err != nil {
		// Error during the transaction
		return backend.TrialsInfoResult{}, backend.NewUnexpectedError("unable retrieve requested trials (%w)", err)
	}

	return backend.TrialsInfoResult{TrialInfos: trialInfos, NextTrialIdx: nextTrialIdx}, nil
}

func (b *boltBackend) ObserveTrials(
	ctx context.Context,
	filter backend.TrialFilter,
	fromTrialIdx int,
	count int,
	out chan<- backend.TrialsInfoResult,
) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		remainingCount := count
		nextTrialIdx := fromTrialIdx
		for {
			partialResult, err := b.RetrieveTrials(ctx, filter, nextTrialIdx, remainingCount)
			if err != nil {
				return err
			}
			retrievedCount := len(partialResult.TrialInfos)
			if retrievedCount > 0 {
				// Some new results to send
				select {
				case <-ctx.Done():
					return ctx.Err()
				case out <- partialResult:
					if remainingCount > 0 {
						remainingCount -= retrievedCount
						if remainingCount <= 0 {
							// Everything requested retrieved
							return nil
						}
					}
					nextTrialIdx = partialResult.NextTrialIdx
				}
			} else {
				// No new results to send
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(b.observeDbPollingDelay):
					continue
				}
			}
		}
	})

	return g.Wait()
}

func (b *boltBackend) DeleteTrials(_ context.Context, trialIDs []string) error {
	err := b.db.Batch(func(tx *bolt.Tx) error {
		// Function must be idempotent as it might be called multiple times
		trialsBucket := getTrialsBucket(tx)
		trialsIdxBucket := getTrialsIdxBucket(tx)
		for _, trialID := range trialIDs {
			// Retrieving this trial's idx
			trialIDKey := serializeTrialID(trialID)
			trialBucket := trialsBucket.Bucket(trialIDKey)

			if trialBucket == nil {
				// The trial already doesn't exist
				continue
			}

			// This is an existing trial, retrieving its idx
			metadataV := trialBucket.Get(metadataKey)
			if metadataV == nil {
				return backend.NewUnexpectedError("no metadata for trial %q", trialID)
			}
			metadata, err := deserializeTrialMetadata(metadataV)
			if err != nil {
				return err
			}
			trialIdx := metadata.TrialIdx

			// Delete the trial bucket
			err = trialsBucket.DeleteBucket(trialIDKey)
			if err != nil {
				return backend.NewUnexpectedError("unable to delete trial %q bucket (%w)", trialID, err)
			}

			// Delete the trial idx
			err = trialsIdxBucket.Delete(serializeNumID(trialIdx))
			if err != nil {
				return backend.NewUnexpectedError("unable to delete trial %q idx (%w)", trialID, err)
			}
		}
		return nil
	})

	if err != nil {
		// Error during the insertion
		return err
	}

	return nil
}

func getTrialParams(tx *bolt.Tx, trialIDs []string) ([]*backend.TrialParams, error) {
	paramsList := []*backend.TrialParams{}
	trialsBucket := getTrialsBucket(tx)

	for _, trialID := range trialIDs {
		trialBucket := trialsBucket.Bucket(serializeTrialID(trialID))
		if trialBucket == nil {
			return []*backend.TrialParams{}, &backend.UnknownTrialError{TrialID: trialID}
		}

		// Retrieving the trial params
		paramsV := trialBucket.Get(paramsKey)
		if paramsV == nil {
			return []*backend.TrialParams{}, backend.NewUnexpectedError("no params for trial %q", trialID)
		}

		params, err := deserializeTrialParams(paramsV)
		if err != nil {
			return []*backend.TrialParams{}, err
		}

		// Retrieving the trial metadata
		trialMetadataV := trialBucket.Get(metadataKey)
		if trialMetadataV == nil {
			return []*backend.TrialParams{}, backend.NewUnexpectedError("no metadata for trial %q", trialID)
		}

		metadata, err := deserializeTrialMetadata(trialMetadataV)
		if err != nil {
			return []*backend.TrialParams{}, err
		}

		paramsList = append(paramsList, &backend.TrialParams{
			TrialID: trialID,
			UserID:  metadata.UserID,
			Params:  params,
		})
	}

	return paramsList, nil
}

func (b *boltBackend) GetTrialParams(_ context.Context, trialIDs []string) ([]*backend.TrialParams, error) {
	paramsList := []*backend.TrialParams{}
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		paramsList, err = getTrialParams(tx, trialIDs)
		return err
	})

	if err != nil {
		// Error during the insertion
		return []*backend.TrialParams{}, err
	}

	return paramsList, nil
}

func (b *boltBackend) AddSamples(_ context.Context, samples []*grpcapi.StoredTrialSample) error {
	err := b.db.Batch(func(tx *bolt.Tx) error {
		// Function must be idempotent as it might be called multiple times
		trialsBucket := getTrialsBucket(tx)

		for _, sample := range samples {
			trialBucket := trialsBucket.Bucket(serializeTrialID(sample.TrialId))
			if trialBucket == nil {
				return &backend.UnknownTrialError{TrialID: sample.TrialId}
			}

			samplesBucket := trialBucket.Bucket(samplesBucketName)
			if samplesBucket == nil {
				return backend.NewUnexpectedError("no sample bucket for trial %q", sample.TrialId)
			}

			sampleValue, err := serializeSample(sample)
			if err != nil {
				return err
			}
			sampleKey := serializeNumID(sample.TickId)

			tickExists := samplesBucket.Get(sampleKey)
			err = samplesBucket.Put(sampleKey, sampleValue)
			if err != nil {
				return backend.NewUnexpectedError(
					"unable to put sample %d for trial %q (%w)",
					sample.TickId, sample.TrialId, err,
				)
			}

			//increment sample count if tick id didn't exist
			if tickExists == nil {
				prevCount := getSampleCount(trialBucket, sample.TrialId)
				err = trialBucket.Put(sampleCountKey, serializeNumID(uint64(prevCount+1)))
				if err != nil {
					return err
				}
			}

		}
		return nil
	})

	if err != nil {
		// Error during the insertion
		return err
	}

	return nil
}

func (b *boltBackend) ObserveSamples(
	ctx context.Context,
	filter backend.TrialSampleFilter,
	out chan<- *grpcapi.StoredTrialSample,
) error {
	paramsList := []*backend.TrialParams{}
	err := b.db.View(func(tx *bolt.Tx) error {
		var err error
		paramsList, err = getTrialParams(tx, filter.TrialIDs)
		return err
	})

	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, params := range paramsList {
		appliedFilter := backend.NewAppliedTrialSampleFilter(filter, params.Params)
		params := params // New 'params' that gets captured by the goroutine's closure
		g.Go(func() error {
			trialEnded := false
			var lastTickIDKey []byte
			for {
				// Retrieving a bunch of samples for this trial
				err := b.db.View(func(tx *bolt.Tx) error {
					// Assuming all the buckets are there
					// We already checked before that the trial exists and the other buckets are "static"
					samplesBucket := getTrialsBucket(tx).Bucket(serializeTrialID(params.TrialID)).Bucket(samplesBucketName)

					var tickIDKey []byte
					var sampleV []byte
					c := samplesBucket.Cursor()
					if lastTickIDKey == nil {
						// No 'saved' key, start at the beginning
						tickIDKey, sampleV = c.First()
					} else {
						// A key has been saved, seeking it
						_, _ = c.Seek(lastTickIDKey)
						// And then go to the following one
						tickIDKey, sampleV = c.Next()
					}
					for ; tickIDKey != nil; tickIDKey, sampleV = c.Next() {
						sample, err := deserializeSample(sampleV)
						if err != nil {
							return err
						}

						trialEnded = sample.State == grpcapi.TrialState_ENDED
						filteredSample := appliedFilter.Filter(sample)

						select {
						case <-ctx.Done():
							return ctx.Err()
						case out <- filteredSample:
							// A valid key was reached, saving it
							lastTickIDKey = make([]byte, len(tickIDKey))
							copy(lastTickIDKey, tickIDKey)
							continue
						}
					}
					return nil
				})
				if err != nil {
					return err
				}
				if trialEnded {
					break
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(b.observeDbPollingDelay):
					continue
				}
			}
			return nil
		})
	}

	return g.Wait()
}
