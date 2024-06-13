package nats

import (
	"context"
	"log"
	"time"

	"github.com/menderdevicesconsumer/internal/device"
	"github.com/nats-io/nats.go/jetstream"
)

func handleRequest(js jetstream.JetStream, msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if msg.Subject() == "user.login.>" {
		creds, _ := device.ParseCredentials(msg)
		_, err := creds.AuthenticateWithContext(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to authenticate: %v", err)
			msg.Ack()
		}
		return

	}

	if msg.Subject() == "device.listDevice.>" {
		_, err := device.GetDeviceList(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to fetch devices: %v", err)
			msg.Ack()
			return
		}
		return
	}

	if msg.Subject() == "device.preauthorizeDevice.>" {
		_, err := device.PreauthorizeDevice(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to fetch devices: %v", err)
			msg.Ack()
			return
		}
		return
	}

	if msg.Subject() == "device.uploadArtifact.>" {
		_, err := device.UploadArtifact(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to upload Artifact: %v", err)
			msg.Ack()
			return
		}
		return
	}
}
