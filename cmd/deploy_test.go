package cmd

import (
	"bytes"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/cogment/cogment/deployment"
	"net/http"
	"testing"
)

type mockedDeployment struct {
	mock.Mock
}

func (m *mockedDeployment) PushImages(manifest *deployment.DeploymentManifest) {
	m.Called(manifest)
}

func TestDeployCommandWithStatusCode(t *testing.T) {

	const appId = "my-app-731841"

	client := resty.New()
	//client.SetDebug(false)

	viper.SetFs(afero.NewMemMapFs())
	initConfig()
	viper.Set("app", appId)

	httpmock.ActivateNonDefault(client.GetClient())
	defer httpmock.DeactivateAndReset()

	manifest, err := deployment.CreateManifestFromCompose("../testdata/docker-compose.yaml", []string{})
	if err != nil {
		t.Fatal(err)
	}

	myErr := make(map[string]interface{})
	myErr["detail"] = "error"

	var tests = []struct {
		statusCode  int
		response    interface{}
		expectedErr error
	}{
		{201, "", nil},
		{400, myErr, fmt.Errorf("{\"detail\":\"error\"}")},
		{401, myErr, fmt.Errorf("{\"detail\":\"error\"}")},
		{404, "invalid", fmt.Errorf("Application not found")},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.statusCode), func(t *testing.T) {

			httpmock.Reset()
			httpmock.RegisterResponder("POST", fmt.Sprintf("/applications/%s/deploy", appId),
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.statusCode, tt.response)
				},
			)
			var stdin bytes.Buffer
			stdin.Write([]byte("y\n"))

			mockDeploy := new(mockedDeployment)

			mockDeploy.On("PushImages", manifest).Return().Once()

			ctx := deployCmdContext{
				imagesPusher: mockDeploy.PushImages,
				client:       client,
				stdin:        &stdin,
			}
			err := runDeployCmd(manifest, &ctx)

			mockDeploy.AssertExpectations(t)
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
