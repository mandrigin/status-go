package main

import (
	"log"
	"time"

	"github.com/status-im/status-go/cmd/bots"
	"github.com/status-im/status-go/geth/api"
)

func main() {
	config, err := bots.NodeConfig()
	if err != nil {
		log.Fatalf("Making config failed: %v", err)
		return
	}

	backend := api.NewStatusBackend()
	log.Println("Starting node...")
	err = backend.StartNode(config)
	if err != nil {
		log.Fatalf("Node start failed: %v", err)
		return
	}

	node, err := backend.NodeManager().Node()
	if err != nil {
		log.Fatalf("Getting node failed: %v", err)
		return
	}

	bots.SignupOrLogin(api.NewStatusAPIWithBackend(backend), "my-cool-password").Join("humans-need-not-apply", "Cloudy Test Baboon").RepeatEvery(100*time.Millisecond, func(ch *bots.StatusChannel) {
		for _, msg := range ch.ReadMessages() {
			log.Println("RCVD:", msg)
		}
	})

	// wait till node has been stopped
	node.Wait()
}
