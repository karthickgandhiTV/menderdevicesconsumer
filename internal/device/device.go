package device

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/menderdevicesconsumer/internal/api"
	httpclient "github.com/menderdevicesconsumer/internal/http"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Request struct {
	RequestId string `json:"requestId"`
	Token     string `json:"token"`
	Domain    string `json:"domain"`
}

type DeviceData struct {
	MAC string `json:"mac"`
	// DyngateId string `json:"dyngateId"`
}

type PreauthorizeDeviceRequest struct {
	DeviceData  DeviceData `json:"identity_data"`
	Pubkey      string     `json:"pubkey"`
	RequestData Request    `json:"request_data"`
}

type AcceptDeviceRequest struct {
	Deviceid    string  `json:"deviceId"`
	AuthsetId   string  `json:"authSetId"`
	RequestData Request `json:"request_data"`
}

func ParseRequest(msg jetstream.Msg) (*Request, error) {
	var request Request
	err := json.Unmarshal(msg.Data(), &request)
	if err != nil {
		log.Printf("Failed to parse request: %v", err)
		return nil, err
	}
	return &request, nil
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

func ParseAcceptDeviceRequest(msg jetstream.Msg) (*AcceptDeviceRequest, error) {
	var acceptDeviceReq AcceptDeviceRequest
	err := json.Unmarshal(msg.Data(), &acceptDeviceReq)
	if err != nil {
		log.Printf("Failed to parse accept device request: %v", err)
		return nil, err
	}
	return &acceptDeviceReq, nil
}

// PerformAPIRequest sends an HTTP request to the specified API endpoint and returns the response body and status code.
func PerformAPIRequest(ctx context.Context, method, apiURL, token string, body io.Reader) ([]byte, error, int) {
	client := httpclient.NewClient()
	req, err := httpclient.NewRequestWithContext(ctx, method, apiURL, body)
	if err != nil {
		return nil, err, 1
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err, 1
	}
	defer resp.Body.Close()

	response, _ := io.ReadAll(resp.Body)

	return response, nil, resp.StatusCode
}

func HandleAPIRequest(ctx context.Context, js jetstream.JetStream, req Request, apiRoute, responseSubject, method string, body io.Reader) (string, error) {
	log.Printf("Received Request: %s", req.RequestId)
	apiURL := "https://" + req.Domain + apiRoute
	resp, err, StatusCode := PerformAPIRequest(ctx, method, apiURL, req.Token, body)
	if err != nil {
		log.Printf("Failed to make API request: %v", err)
		return "", err
	}
	log.Print(string(resp))
	log.Print(responseSubject + req.RequestId)
	responseMsg := nats.NewMsg(responseSubject + req.RequestId)
	responseMsg.Header.Set("StatusCode", fmt.Sprint(StatusCode))
	responseMsg.Data = append(responseMsg.Data, resp...)
	fmt.Println(responseMsg.Header.Get("StatusCode"))
	_, err = js.PublishMsgAsync(responseMsg)
	log.Print(err)
	if err != nil {
		log.Printf("Failed to publish : %v", err)
	}
	return string(resp), nil
}

func GetDeviceList(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	apiConfig := api.GetConfig()
	req, _ := ParseRequest(msg)

	return HandleAPIRequest(ctx, js, *req, apiConfig.API.V2uriDevices, "device.listDeviceResponse.", "GET", nil)
}

func PreauthorizeDevice(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	request, err := ParseDeviceInfoRequest(msg)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
	}
	apiConfig := api.GetConfig()

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

	return HandleAPIRequest(ctx, js, request.RequestData, apiConfig.API.V2uriDevices, "device.preauthorizeDeviceResponse.", "POST", identityDataReader)
}

func AcceptDevice(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	request, err := ParseAcceptDeviceRequest(msg)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
	}

	apiConfig := api.GetConfig()

	payload := struct {
		Status string `json:"status"`
	}{
		Status: "accepted",
	}

	statusData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to parse status data: %v", err)
		return "", err
	}

	statusDataReader := bytes.NewReader(statusData)

	return HandleAPIRequest(ctx, js, request.RequestData, apiConfig.API.V2uriDevices+"/"+request.Deviceid+"/auth/"+request.AuthsetId+"/status", "device.acceptDeviceResponse.", "PUT", statusDataReader)

}

func RejectDevice(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	request, err := ParseAcceptDeviceRequest(msg)
	if err != nil {
		log.Printf("Unable to parse request: %v", err)
	}

	apiConfig := api.GetConfig()

	payload := struct {
		Status string `json:"status"`
	}{
		Status: "rejected",
	}

	statusData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to parse status data: %v", err)
		return "", err
	}

	statusDataReader := bytes.NewReader(statusData)

	return HandleAPIRequest(ctx, js, request.RequestData, apiConfig.API.V2uriDevices+"/"+request.Deviceid+"/auth/"+request.AuthsetId+"/status", "device.rejectDeviceResponse.", "PUT", statusDataReader)

}
