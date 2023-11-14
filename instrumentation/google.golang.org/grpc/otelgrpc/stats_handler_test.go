package otelgrpc

import (
	"context"
	"sync"
	"testing"

	"google.golang.org/grpc/stats"
)

func TestHandleRPC(t *testing.T) {
	const iteration = 100
	wg := &sync.WaitGroup{}
	ctx := context.Background()
	h := NewClientHandler()
	ctx = h.TagRPC(ctx, &stats.RPCTagInfo{
		FullMethodName: "test/method",
	})
	for i := 0; i < iteration; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.HandleRPC(ctx, &stats.Begin{})
			wg.Add(3)
			go func() {
				defer wg.Done()
				h.HandleRPC(ctx, &stats.InPayload{})
			}()
			go func() {
				defer wg.Done()
				h.HandleRPC(ctx, &stats.OutPayload{})
			}()
			go func() {
				defer wg.Done()
				h.HandleRPC(ctx, &stats.End{})
			}()
		}()
	}
	wg.Wait()
}
