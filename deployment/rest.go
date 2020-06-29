package deployment

import (
	"encoding/json"
	"errors"
	"github.com/go-resty/resty/v2"
	"gitlab.com/cogment/cogment/helper"
)

func PlatformClient(verbose bool) (*resty.Client, error) {
	baseURL := helper.CurrentConfig("url")
	token := helper.CurrentConfig("token")

	if baseURL == "" {
		return nil, errors.New("API URL is not defined, maybe try `cogment configure remote`")
	}

	client := resty.New()
	client.SetHostURL(baseURL)
	client.SetHeader("Authorization", "Token "+token)
	//client.SetAuthToken(token)
	client.SetDebug(verbose)

	// Registering global Error object structure for JSON/XML request
	//client.SetError(Error{}) // or resty.SetError(Error{})

	return client, nil
}

func ResponseFormat(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}
