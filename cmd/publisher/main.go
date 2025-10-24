package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	stan "github.com/nats-io/stan.go"
)

func main() {
	var (
		file      = flag.String("f", "model.json", "path to JSON payload")
		clusterID = flag.String("cluster", "test-cluster", "NATS Streaming cluster ID")
		clientID  = flag.String("client", "publisher", "NATS Streaming client ID")
		subject   = flag.String("subject", "orders", "subject to publish to")
		natsURL   = flag.String("url", "nats://localhost:4222", "NATS Streaming server URL")
	)
	flag.Parse()

	payload, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("read payload: %v", err)
	}

	if !json.Valid(payload) {
		log.Fatal("file does not contain valid JSON")
	}

	sc, err := stan.Connect(*clusterID, fmt.Sprintf("%s-%d", *clientID, randInt()), stan.NatsURL(*natsURL))
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer sc.Close()

	if err := sc.Publish(*subject, payload); err != nil {
		log.Fatalf("publish: %v", err)
	}

	log.Printf("published %d bytes to %s", len(payload), *subject)
}

func randInt() int64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<31))
	if err != nil {
		panic(err)
	}
	return n.Int64()
}
