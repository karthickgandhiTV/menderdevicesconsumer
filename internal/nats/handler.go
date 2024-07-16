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

	log.Print(string(msg.Data()))

	switch msg.Subject() {
	case "user.login.>":
		creds, _ := device.ParseCredentials(msg)
		_, err := creds.AuthenticateWithContext(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to authenticate: %v", err)
			msg.Ack()
		}

	case "device.listDevice.>":
		_, err := device.GetDeviceList(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to fetch devices: %v", err)
			msg.Ack()

		}

	case "device.preauthorizeDevice.>":
		_, err := device.PreauthorizeDevice(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to fetch devices: %v", err)
			msg.Ack()

		}

	case "device.acceptDevice.>":
		_, err := device.AcceptDevice(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to accept device: %v", err)
			msg.Ack()

		}

	case "device.rejectDevice.>":
		_, err := device.RejectDevice(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to accept device: %v", err)
			msg.Ack()

		}

	}

}
