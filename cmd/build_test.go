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

package cmd

//func TestParseServicesFromCompose_ErrorHandling(t *testing.T) {
//		cases := []struct {
//			fixture   string
//			returnErr bool
//			name      string
//		}{
//			{
//				fixture:   "testdata/docker-compose.yml",
//				returnErr: false,
//				name:      "EmptyFile",
//			},
//			{
//				fixture:   "testdata/docker-compose.yml",
//				returnErr: false,
//				name:      "EmptyFile",
//			},
//			{
//				fixture:   "testdata/grades/invalid.csv",
//				returnErr: true,
//				name:      "InvalidFile",
//			},
//			{
//				fixture:   "testdata/grades/nonexisting.csv",
//				returnErr: true,
//				name:      "NonexistingFile",
//			},
//			{
//				fixture:   "testdata/grades/valid.csv",
//				returnErr: false,
//				name:      "ValidFile",
//			},
//		}
//
//}

//func TestNewGradebook_ErrorHandling(t *testing.T) {
//	cases := []struct {
//		fixture   string
//		returnErr bool
//		name      string
//	}{
//		{
//			fixture:   "testdata/docker-compose.yml",
//			returnErr: false,
//			name:      "EmptyFile",
//		},
//		{
//			fixture:   "testdata/grades/invalid.csv",
//			returnErr: true,
//			name:      "InvalidFile",
//		},
//		{
//			fixture:   "testdata/grades/nonexisting.csv",
//			returnErr: true,
//			name:      "NonexistingFile",
//		},
//		{
//			fixture:   "testdata/grades/valid.csv",
//			returnErr: false,
//			name:      "ValidFile",
//		},
//	}
//
//	for _, tc := range cases {
//		t.Run(tc.name, func(t *testing.T) {
//			_, err := NewGradebook(tc.fixture)
//			returnedErr := err != nil
//
//			if returnedErr != tc.returnErr {
//				t.Fatalf("Expected returnErr: %v, got: %v", tc.returnErr, returnedErr)
//			}
//		})
//	}
//}
