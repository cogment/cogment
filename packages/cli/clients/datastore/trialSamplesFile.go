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

package datastore

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	cogmentAPI "github.com/cogment/cogment/grpcapi/cogment/api"
	"github.com/cogment/cogment/utils"
	"github.com/cogment/cogment/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const magicValue = "COGMENTTRIALS001"

type state int

const (
	expectHeader state = iota
	expectSamples
	failure
)

type TrialSamplesFileWriter struct {
	writer io.Writer
	state  state
	Bytes  int
}

func CreateTrialSamplesFileWriter(writer io.Writer) *TrialSamplesFileWriter {
	return &TrialSamplesFileWriter{
		writer: writer,
		state:  expectHeader,
		Bytes:  0,
	}
}

func (writer *TrialSamplesFileWriter) WriteHeader(trialParams map[string]*cogmentAPI.TrialParams) error {
	if writer.state != expectHeader {
		return fmt.Errorf("writer not in the expected state")
	}

	bytesWritten, err := writeMagicValue(writer.writer, magicValue)
	writer.Bytes += bytesWritten
	if err != nil {
		writer.state = failure
		return err
	}

	header := &cogmentAPI.TrialSamplesFileHeader{
		VersionInfo: &cogmentAPI.VersionInfo{
			Versions: []*cogmentAPI.VersionInfo_Version{
				{
					Name:    "cogment",
					Version: version.Version,
				},
			},
		},
		ExportTimestamp: uint64(time.Now().UnixNano()),
		TrialParams:     trialParams,
	}

	bytesWritten, err = writeProtobufMessage(writer.writer, header)
	writer.Bytes += bytesWritten
	if err != nil {
		writer.state = failure
		return err
	}
	writer.state = expectSamples
	return nil
}

func (writer *TrialSamplesFileWriter) WriteSample(trialSample *cogmentAPI.StoredTrialSample) error {
	if writer.state != expectSamples {
		return fmt.Errorf("writer not in the expected state")
	}

	bytesWritten, err := writeProtobufMessage(writer.writer, trialSample)
	writer.Bytes += bytesWritten
	if err != nil {
		writer.state = failure
		return err
	}
	return nil
}

type TrialSamplesFileReader struct {
	reader io.Reader
	state  state
}

func CreateTrialSamplesFileReader(reader io.Reader) *TrialSamplesFileReader {
	return &TrialSamplesFileReader{
		reader: reader,
		state:  expectHeader,
	}
}

func (reader *TrialSamplesFileReader) ReadHeader() (*cogmentAPI.TrialSamplesFileHeader, error) {
	if reader.state != expectHeader {
		return nil, fmt.Errorf("reader not in the expected state")
	}
	err := readMagicValue(reader.reader, magicValue)
	if err != nil {
		reader.state = failure
		return nil, err
	}

	header := &cogmentAPI.TrialSamplesFileHeader{}
	err = readProtobufMessage(reader.reader, header)
	if err != nil {
		reader.state = failure
		return nil, err
	}
	reader.state = expectSamples
	return header, nil
}

func (reader *TrialSamplesFileReader) ReadSample() (*cogmentAPI.StoredTrialSample, error) {
	if reader.state != expectSamples {
		return nil, fmt.Errorf("reader not in the expected state")
	}

	trialSample := &cogmentAPI.StoredTrialSample{}
	err := readProtobufMessage(reader.reader, trialSample)
	if err != nil {
		reader.state = failure
		return nil, err
	}
	return trialSample, err
}

func writeMagicValue(writer io.Writer, magicValue string) (int, error) {
	writtenSize := 0
	magicSize, err := writer.Write([]byte(magicValue))
	writtenSize += magicSize
	if err != nil {
		return writtenSize, err
	}
	return writtenSize, err
}

func readMagicValue(reader io.Reader, expectedMagicValue string) error {
	expectedSize := len(expectedMagicValue)
	readMagicValue := make([]byte, expectedSize)
	readSize, err := reader.Read(readMagicValue)
	if err != nil {
		return err
	}
	if readSize != expectedSize {
		return fmt.Errorf(
			"read [%d / %s] bytes, expected [%d / %s] bytes",
			readSize, utils.FormatBytes(readSize),
			expectedSize, utils.FormatBytes(expectedSize),
		)
	}
	if string(readMagicValue) != expectedMagicValue {
		return fmt.Errorf("magic value doesn't match, expected [%s], got [%s]", expectedMagicValue, readMagicValue)
	}
	return nil
}

func writeProtobufMessage(writer io.Writer, message protoreflect.ProtoMessage) (int, error) {
	writtenSize := 0
	serializedMessage, err := proto.Marshal(message)
	if err != nil {
		return writtenSize, err
	}
	size := uint32(len(serializedMessage))
	err = binary.Write(writer, binary.LittleEndian, &size)
	if err != nil {
		return writtenSize, err
	}
	writtenSize += binary.Size(size)
	messageSize, err := writer.Write(serializedMessage)
	writtenSize += messageSize
	return writtenSize, err
}

func readProtobufMessage(reader io.Reader, message protoreflect.ProtoMessage) error {
	var size uint32
	err := binary.Read(reader, binary.LittleEndian, &size)
	if err != nil {
		return err
	}

	pbBuffer := make([]byte, size)
	readSize, err := reader.Read(pbBuffer)
	if err != nil {
		return err
	}
	expectedSize := int(size)
	if readSize != int(size) {
		return fmt.Errorf(
			"read [%d / %s] bytes, expected [%d / %s] bytes",
			readSize, utils.FormatBytes(readSize),
			expectedSize, utils.FormatBytes(expectedSize),
		)
	}
	return proto.Unmarshal(pbBuffer, message)
}
