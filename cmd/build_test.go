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
