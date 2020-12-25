// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelttrpc

// ttrpc tracing middleware
// https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/trace/semantic_conventions/rpc.md
import (
	"context"
	"fmt"
	"strings"

	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/containerd/ttrpc"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	otelcontrib "go.opentelemetry.io/contrib"
)

const protocolName = "ttrpc"

// UnaryClientInterceptor returns a ttrpc.UnaryClientInterceptor suitable
// for use in a ttrpc.NewClient call.
func UnaryClientInterceptor(opts ...Option) ttrpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		req *ttrpc.Request,
		resp *ttrpc.Response,
		ci *ttrpc.UnaryClientInfo,
		invoker ttrpc.Invoker,
	) error {
		requestMetadata, ok := ttrpc.GetMetadata(ctx)
		if !ok {
			requestMetadata = make(ttrpc.MD)
		}

		tracer := newConfig(opts).TracerProvider.Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(otelcontrib.SemVersion()),
		)

		name, attr := clientSpanInfo(req)
		var span trace.Span
		ctx, span = tracer.Start(
			ctx,
			name,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		Inject(ctx, &requestMetadata, opts...)
		ctx = ttrpc.WithMetadata(ctx, requestMetadata)

		setRequest(req, &requestMetadata)
		err := invoker(ctx, req, resp)

		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(statusCodeAttr(s.Code()))
		} else {
			span.SetAttributes(statusCodeAttr(grpc_codes.OK))
		}

		return err
	}
}

func setRequest(req *ttrpc.Request, md *ttrpc.MD) {
	newMD := make([]*ttrpc.KeyValue, 0)
	for _, kv := range req.Metadata {
		// not found in md, means that we can copy old kv
		// otherwise, we will use the values in md to overwrite it
		if _, found := md.Get(kv.Key); !found {
			newMD = append(newMD, kv)
		}
	}

	req.Metadata = newMD

	for k, values := range *md {
		for _, v := range values {
			req.Metadata = append(req.Metadata, &ttrpc.KeyValue{
				Key:   k,
				Value: v,
			})
		}
	}
}

// UnaryServerInterceptor returns a ttrpc.UnaryServerInterceptor suitable
// for use in a ttrpc.NewServer call.
func UnaryServerInterceptor(opts ...Option) ttrpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		um ttrpc.Unmarshaler,
		si *ttrpc.UnaryServerInfo,
		m ttrpc.Method,
	) (interface{}, error) {
		requestMetadata, ok := ttrpc.GetMetadata(ctx)
		if !ok {
			requestMetadata = make(ttrpc.MD)
		}

		entries, spanCtx := Extract(ctx, &requestMetadata, opts...)

		ctx = baggage.ContextWithValues(ctx, entries...)

		tracer := newConfig(opts).TracerProvider.Tracer(
			instrumentationName,
			trace.WithInstrumentationVersion(otelcontrib.SemVersion()),
		)

		name, attr := serverSpanInfo(si.FullMethod)
		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, spanCtx),
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		resp, err := m(ctx, um)
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(statusCodeAttr(s.Code()))
		} else {
			span.SetAttributes(statusCodeAttr(grpc_codes.OK))
		}

		return resp, err
	}
}

// clientSpanInfo returns a span name and all appropriate attributes from
// the ttrpc request
func clientSpanInfo(req *ttrpc.Request) (string, []label.KeyValue) {
	return fmt.Sprintf("%s/%s", req.Service, req.Method), []label.KeyValue{
		semconv.RPCServiceKey.String(req.Service),
		semconv.RPCMethodKey.String(req.Method),
		semconv.MessagingMessagePayloadSizeBytesKey.Int(len(req.Payload)),
		semconv.RPCSystemGRPC,
		semconv.MessagingProtocolKey.String(protocolName),
	}
}

// serverSpanInfo returns a span name and all appropriate attributes from
// the ttrpc request
func serverSpanInfo(fullMethod string) (string, []label.KeyValue) {
	name := strings.TrimLeft(fullMethod, "/")
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		// Invalid format, does not follow `/package.service/method`.
		return name, []label.KeyValue(nil)
	}

	return name, []label.KeyValue{
		semconv.RPCServiceKey.String(parts[0]),
		semconv.RPCMethodKey.String(parts[1]),
		semconv.RPCSystemGRPC,
		semconv.MessagingProtocolKey.String(protocolName),
	}
}

// statusCodeAttr returns status code attribute based on given gRPC code
func statusCodeAttr(c grpc_codes.Code) label.KeyValue {
	return TTRPCStatusCodeKey.Uint32(uint32(c))
}
