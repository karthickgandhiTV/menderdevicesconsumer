package device

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"

	"github.com/menderdevicesconsumer/internal/api"
	"github.com/menderdevicesconsumer/internal/auth"
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
	// api := api.GetInstance()
	client := http.NewClient()

	apiURL := "https://" + creds.Domain + "/api/management/v1/deployments/artifacts/directupload"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create API request: %v", err)
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
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

	_, err = js.Publish(ctx, "device.listDeviceResponse."+creds.RequestId, []byte(body))
	if err != nil {
		log.Printf("Failed to publish response%v", err)
	}

	return string(body), nil
}

type DeviceData struct {
	MAC string `json:"mac"`
}

type PreauthorizeDeviceRequest struct {
	DeviceData  DeviceData   `json:"identity_data"`
	Pubkey      string       `json:"pubkey"`
	RequestData auth.Request `json:"request_data"`
}

func ParseDeviceInfoRequest(msg jetstream.Msg) (*PreauthorizeDeviceRequest, error) {
	var deviceInfoReq PreauthorizeDeviceRequest
	err := json.Unmarshal(msg.Data(), &deviceInfoReq)
	if err != nil {
		log.Printf("Failed to parse device info request: %v", err)
		return nil, err
	}
	return &deviceInfoReq, nil
}

func PreauthorizeDevice(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	request, err := ParseDeviceInfoRequest(msg)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return "", err
	}

	log.Print(request.RequestData.Domain)
	log.Printf("Received Request: %s", request.RequestData.RequestId)
	token, err := request.RequestData.AuthenticateWithContext(ctx)
	if err != nil {
		log.Printf("Failed to authenticate: %v", err)
		return "", err
	}
	api := api.GetInstance()
	client := http.NewClient()

	payload := struct {
		DeviceData DeviceData `json:"identity_data"`
		Pubkey     string     `json:"pubkey"`
	}{
		DeviceData: request.DeviceData,
		Pubkey:     request.Pubkey,
	}

	deviceData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to parse device data: %v", err)
		return "", err
	}

	identityDataReader := bytes.NewReader(deviceData)

	apiURL := "https://" + request.RequestData.Domain + api.API.V2uriDevices
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, identityDataReader)
	if err != nil {
		log.Printf("Failed to create API request: %v", err)
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
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

	_, err = js.Publish(ctx, "device.preauthorizeDeviceResponse."+request.RequestData.RequestId, []byte(body))
	if err != nil {
		log.Printf("Failed to publish response%v", err)
	}
	log.Print(resp.Header.Get("Location"))
	log.Print(resp.Status)
	log.Print(string(body))

	return string(body), nil
}
