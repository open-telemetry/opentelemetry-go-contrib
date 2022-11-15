package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/contrib/instrumentation/github.com/nats-io/nats.go/otelnats"
)

const subject = "subject"

func exit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx := context.Background()

	nc, err := nats.Connect("otelnats-nats:4222")
	exit(err)

	nc.Subscribe(subject, func(msg *nats.Msg) {
		span := otelnats.SpanFrom(ctx, msg)
		defer span.End()

		spanCtx := span.SpanContext()
		traceID := spanCtx.TraceID().String()
		spanID := spanCtx.SpanID().String()

		fmt.Printf("created span from msg with parent TraceID %s SpanID %s\n", traceID, spanID)
	})

	done := make(chan os.Signal)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	<-done
}
