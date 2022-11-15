package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/nats-io/nats.go/otelnats"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const subject = "subject"

func exit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	parent := context.Background()

	nc, err := nats.Connect("otelnats-nats:4222")
	exit(err)

	tracer := sdktrace.NewTracerProvider().Tracer("tracer")

	for {
		select {
		case <-time.After(time.Millisecond * 500):
			ctx, span := tracer.Start(parent, "publish")

			msg := otelnats.NewMsg(ctx)
			msg.Subject = subject
			if err = nc.PublishMsg(msg); err != nil {
				log.Println("publish err", err)
			}
			spanCtx := span.SpanContext()
			traceID := spanCtx.TraceID().String()
			spanID := spanCtx.SpanID().String()
			fmt.Printf("published msg with TraceID %s SpanID %s\n", traceID, spanID)
			span.End()
		}
	}
}
