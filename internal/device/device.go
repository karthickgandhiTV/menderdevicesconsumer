package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"github.com/menderdevicesconsumer/internal/api"
	"github.com/menderdevicesconsumer/internal/auth"
	"github.com/menderdevicesconsumer/internal/http"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
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

func UploadArtifact(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	containerName := "karthick"
	blobName := "core-image-base-raspberrypi3-20240513103357.mender"

	request, err := ParseCredentials(msg)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return "", err
	}

	log.Print(request.Domain)
	log.Printf("Received Request: %s", request.RequestId)
	token, err := request.AuthenticateWithContext(ctx)
	if err != nil {
		log.Printf("Failed to authenticate: %v", err)
		return "", err
	}

	url := "https://menderartifactstorage.blob.core.windows.net/"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("Failed to crate default Credential: %v", err)
		return "", err
	}

	serviceClient, err := azblob.NewClient(url, credential, nil)
	if err != nil {
		log.Printf("Failed to create service client: %v", err)
		return "", err
	}

	downloadResponse, err := serviceClient.DownloadStream(ctx, containerName, blobName, nil)
	if err != nil {
		log.Printf("Failed to start blob download: %v", err)
		return "", err
	}

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)

	go func() {
		defer writer.Close()
		defer downloadResponse.Body.Close()

		metaPart, err := multipartWriter.CreateFormField("description")
		if err != nil {
			log.Printf("Failed to create metadata part: %v", err)
			writer.CloseWithError(err)
			return
		}
		_, err = metaPart.Write([]byte("Your artifact description here"))
		if err != nil {
			log.Printf("Failed to write to metadata part: %v", err)
			writer.CloseWithError(err)
			return
		}

		artifactPart, err := multipartWriter.CreateFormFile("artifact", blobName)
		if err != nil {
			log.Printf("Failed to create form file for artifact: %v", err)
			writer.CloseWithError(err)
			return
		}

		if _, err = io.Copy(artifactPart, downloadResponse.Body); err != nil {
			log.Printf("Failed to copy blob data to form file: %v", err)
			writer.CloseWithError(err)
			return
		}

		if err := multipartWriter.Close(); err != nil {
			log.Printf("Failed to close multipart writer: %v", err)
			writer.CloseWithError(err)
			return
		}
	}()

	client := http.NewClient()
	apiURL := "https://" + request.Domain + "/api/management/v1/deployments/artifacts"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, reader)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", err
	}

	// Set the content type with the correct boundary
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body for debugging
	responseBody, _ := io.ReadAll(resp.Body)
	log.Printf("Server response: %s", responseBody)

	if resp.StatusCode != 201 {
		log.Printf("Failed to upload blob, server responded with status: %v", resp.StatusCode)
		return "", fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	return "Blob uploaded successfully", nil
}
