package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/menderdevicesconsumer/internal/api"
	"github.com/menderdevicesconsumer/internal/http"
)

type Request struct {
	RequestId string `json:"requestId"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Domain    string `json:"domain"`
}

func (creds Request) AuthenticateWithContext(ctx context.Context) (string, error) {
	encodedCreds := base64.StdEncoding.EncodeToString([]byte(creds.Email + ":" + creds.Password))
	headers := map[string][]string{
		"Content-Type":  {"application/json"},
		"Accept":        {"application/jwt"},
		"Authorization": {"Basic " + encodedCreds},
	}
	client := http.NewClient()
	api := api.GetInstance()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://"+creds.Domain+api.API.AuthLogin, nil)
	for key, value := range headers {
		req.Header[key] = value
	}
	if err != nil {
		return "", fmt.Errorf("failed to create API request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send API request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
