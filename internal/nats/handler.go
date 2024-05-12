package nats

import (
	"context"
	"log"
	"time"

	"github.com/menderdevicesconsumer/internal/device"
	"github.com/nats-io/nats.go/jetstream"
)

func handleRequest(js jetstream.JetStream, msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
}
