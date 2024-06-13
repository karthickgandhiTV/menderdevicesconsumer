package device

import (
	"encoding/json"
	"log"

	"github.com/menderdevicesconsumer/internal/auth"
	"github.com/nats-io/nats.go/jetstream"
)

func ParseCredentials(msg jetstream.Msg) (*auth.LoginRequest, error) {
	var creds auth.LoginRequest
	err := json.Unmarshal(msg.Data(), &creds)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return nil, err
	}
	return &creds, nil
}
