// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"os"
	"time"

	"go.opencensus.io/examples/exporter"
	pb "go.opencensus.io/examples/grpc/proto"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	log.Println("Configuring OpenCensus, and registering the Print exporter.")
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
	trace.RegisterExporter(&exporter.PrintExporter{})

	// Set up a connection to the server with the OpenCensus
	// stats handler to enable tracing.
	conn, err := grpc.NewClient(address, grpc.WithStatsHandler(&ocgrpc.ClientHandler{}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	for {
		r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
		if err != nil {
			log.Printf("Could not greet: %v", err)
		} else {
			log.Printf("Greeting: %s", r.Message)
		}
		time.Sleep(2 * time.Second)
	}
}
