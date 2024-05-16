package integrationtests

import (
	"context"
	"net/http"
)

func executeRequest(methodType, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), methodType, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return response, nil
}
