/*
   Copyright The containerd Authors.
   Copyright The OpenTelemetry Authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	context "context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	ttrpc "github.com/containerd/ttrpc"
	"github.com/gogo/protobuf/types"

	"go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc"
	"go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc/example"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
)

const socket = "/tmp/example-ttrpc-server"

func main() {
	flush := initTracer()
	defer flush()

	rand.Seed(time.Now().UnixNano())

	if err := handle(); err != nil {
		log.Fatal(err)
	}
}

func handle() error {
	command := os.Args[1]
	switch command {
	case "server":
		return server()
	case "client":
		return client()
	default:
		return errors.New("invalid command")
	}
}

func server() error {
	s, err := ttrpc.NewServer(
		ttrpc.WithServerHandshaker(ttrpc.UnixSocketRequireSameUser()),
		ttrpc.WithUnaryServerInterceptor(otelttrpc.UnaryServerInterceptor()),
	)
	if err != nil {
		return err
	}
	defer s.Close()
	example.RegisterExampleService(s, &exampleServer{})

	if err := os.Remove(socket); err != nil {
		return err
	}

	l, err := net.Listen("unix", socket)
	if err != nil {
		return err
	}
	defer func() {
		l.Close()
		os.Remove(socket)
	}()
	return s.Serve(context.Background(), l)
}

func client() error {
	return call(func(client example.ExampleService) error {
		r := &example.Method1Request{
			Foo: os.Args[2],
			Bar: os.Args[3],
		}

		ctx := context.Background()
		md := ttrpc.MD{}
		md.Set("name", "koye")
		ctx = ttrpc.WithMetadata(ctx, md)

		// STEP 1: this is the entry point of this example, we will initiate an RPC call
		// from client.
		// this will call `Method1`
		// and will generate two spans: one from the client and one from the server.
		resp, err := client.Method1(ctx, r)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	})
}

type exampleServer struct {
}

func dummyOperation(ctx context.Context) {
	// STEP 4: call RPC method Method2 and generat two new spans:
	tracer := otel.Tracer("billing_app")
	_, span := tracer.Start(ctx, "dummy_operation")
	span.AddEvent("This is a test event", trace.WithAttributes(label.Int("bonus", 996)))
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	defer span.End()
}

func (s *exampleServer) Method1(ctx context.Context, r *example.Method1Request) (*example.Method1Response, error) {
	_ = serverClient(ctx)

	// and then call dummyOperation, this will generate another span that having the same level with serverClient
	dummyOperation(ctx)

	return &example.Method1Response{
		Foo: r.Foo,
		Bar: r.Bar,
	}, nil
}

func (s *exampleServer) Method2(ctx context.Context, r *example.Method1Request) (*types.Empty, error) {
	return &types.Empty{}, nil
}

func serverClient(ctx context.Context) error {
	// STEP 2: in Method1, we will first call serverClient, this will generate span named serverClient
	tracer := otel.Tracer("billing_app")
	ctx, span := tracer.Start(ctx, "server_client")
	defer span.End()

	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	return call(func(client example.ExampleService) error {
		r := &example.Method1Request{
			Foo: "foo",
			Bar: "bar",
		}

		// STEP 3: call RPC method Method2 and generat two new spans:
		// one from client and one from server
		resp, err := client.Method2(ctx, r)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	})
}

func call(f func(client example.ExampleService) error) error {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return err
	}
	defer conn.Close()

	tc := ttrpc.NewClient(conn, ttrpc.WithUnaryClientInterceptor(otelttrpc.UnaryClientInterceptor()))
	client := example.NewExampleClient(tc)

	return f(client)
}
