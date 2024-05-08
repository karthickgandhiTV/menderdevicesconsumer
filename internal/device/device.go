package device

import (
	"context"
	"io"
	"log"

	"github.com/menderdevicesconsumer/internal/api"
	"github.com/menderdevicesconsumer/internal/http"
	"github.com/nats-io/nats.go/jetstream"
)

func GetDeviceList(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	creds, err := ParseCredentials(msg)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return "", err
	}

	log.Print(creds.Domain)
	log.Printf("Received Request: %s", creds.RequestId)
	token, err := creds.AuthenticateWithContext(ctx)
	if err != nil {
		log.Printf("Failed to authenticate: %v", err)
		return "", err
	}
	api := api.GetInstance()
	client := http.NewClient()

	apiURL := "https://" + creds.Domain + api.API.V2uriDevices
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create API request: %v", err)
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send API request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read API response body: %v", err)
		return "", err
	}

	log.Print(string(body))
	_, err = js.Publish(ctx, "device.listDeviceResponse."+creds.RequestId, []byte(body))
	if err != nil {
		log.Printf("Failed to publish response%v", err)
	}

	return string(body), nil
}
