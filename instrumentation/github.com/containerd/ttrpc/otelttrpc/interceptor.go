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

package otelttrpc // import "go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc"

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/containerd/ttrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/github.com/containerd/ttrpc/otelttrpc/internal"
)

type messageType attribute.KeyValue

// Event adds an event of the messageType to the span associated with the
// passed context with a message id.
func (m messageType) Event(ctx context.Context, id int, _ interface{}) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	span.AddEvent("message", trace.WithAttributes(
		attribute.KeyValue(m),
		RPCMessageIDKey.Int(id),
	))
}

var (
	messageSent     = messageType(RPCMessageTypeSent)
	messageReceived = messageType(RPCMessageTypeReceived)
)

// UnaryClientInterceptor returns a ttrpc.UnaryClientInterceptor suitable
// for use in a ttrpc.NewClient call.
func UnaryClientInterceptor(opts ...Option) ttrpc.UnaryClientInterceptor {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	)

	return func(
		ctx context.Context,
		req *ttrpc.Request, reply *ttrpc.Response,
		info *ttrpc.UnaryClientInfo,
		invoker ttrpc.Invoker,
	) error {
		i := &InterceptorInfo{
			Method: info.FullMethod,
			Type:   UnaryClient,
		}
		if cfg.Filter != nil && !cfg.Filter(i) {
			return invoker(ctx, req, reply)
		}

		name, attr := spanInfo(info.FullMethod, "") // TODO(klihub): we can't get the 'target'
		var span trace.Span
		ctx, span = tracer.Start(
			ctx,
			name,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		ctx = inject(ctx, cfg.Propagators)

		messageSent.Event(ctx, 1, req)

		err := invoker(ctx, req, reply)

		messageReceived.Event(ctx, 1, reply)

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

// UnaryServerInterceptor returns ttrpc.UnaryServerInterceptor suitable
// for use in a ttrpc.NewServer call.
func UnaryServerInterceptor(opts ...Option) ttrpc.UnaryServerInterceptor {
	cfg := newConfig(opts)
	tracer := cfg.TracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(Version()),
	)

	return func(
		ctx context.Context,
		unmarshal ttrpc.Unmarshaler, info *ttrpc.UnaryServerInfo, method ttrpc.Method) (interface{}, error) {
		i := &InterceptorInfo{
			Method:          info.FullMethod,
			UnaryServerInfo: info,
			Type:            UnaryServer,
		}
		if cfg.Filter != nil && !cfg.Filter(i) {
			return method(ctx, unmarshal)
		}

		ctx = extract(ctx, cfg.Propagators)

		name, attr := spanInfo(info.FullMethod, peerFromCtx(ctx))
		ctx, span := tracer.Start(
			trace.ContextWithRemoteSpanContext(ctx, trace.SpanContextFromContext(ctx)),
			name,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attr...),
		)
		defer span.End()

		messageReceived.Event(ctx, 1, struct{}{} /*req*/) // TODO(klihub)

		var statusCode grpc_codes.Code
		defer func(t time.Time) {
			elapsedTime := time.Since(t) / time.Millisecond
			attr = append(attr, TTRPCStatusCodeKey.Int64(int64(statusCode)))
			o := metric.WithAttributes(attr...)
			cfg.rpcServerDuration.Record(ctx, int64(elapsedTime), o)
		}(time.Now())

		resp, err := method(ctx, unmarshal)
		if err != nil {
			s, _ := status.FromError(err)
			statusCode, msg := serverStatus(s)
			span.SetStatus(statusCode, msg)
			span.SetAttributes(statusCodeAttr(s.Code()))
			messageSent.Event(ctx, 1, s.Proto())
		} else {
			statusCode = grpc_codes.OK
			span.SetAttributes(statusCodeAttr(grpc_codes.OK))
			messageSent.Event(ctx, 1, resp)
		}

		return resp, err
	}

}

// spanInfo returns a span name and all appropriate attributes from the ttRPC
// method and peer address.
func spanInfo(fullMethod, peerAddress string) (string, []attribute.KeyValue) {
	attrs := []attribute.KeyValue{RPCSystemTTRPC}
	name, mAttrs := internal.ParseFullMethod(fullMethod)
	attrs = append(attrs, mAttrs...)
	attrs = append(attrs, peerAttr(peerAddress)...)
	return name, attrs
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) []attribute.KeyValue {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	var attr []attribute.KeyValue
	if ip := net.ParseIP(host); ip != nil {
		attr = []attribute.KeyValue{
			semconv.NetSockPeerAddr(host),
			semconv.NetSockPeerPort(port),
		}
	} else {
		attr = []attribute.KeyValue{
			semconv.NetPeerName(host),
			semconv.NetPeerPort(port),
		}
	}

	return attr
}

// peerFromCtx returns a peer address from a context, if one exists.
func peerFromCtx(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	return p.Addr.String()
}

// statusCodeAttr returns status code attribute based on given ttRPC code.
func statusCodeAttr(c grpc_codes.Code) attribute.KeyValue {
	return TTRPCStatusCodeKey.Int64(int64(c))
}

// serverStatus returns a span status code and message for a given gRPC
// status code. It maps specific gRPC status codes to a corresponding span
// status code and message. This function is intended for use on the server
// side of a gRPC connection.
//
// If the gRPC status code is Unknown, DeadlineExceeded, Unimplemented,
// Internal, Unavailable, or DataLoss, it returns a span status code of Error
// and the message from the gRPC status. Otherwise, it returns a span status
// code of Unset and an empty message.
func serverStatus(rpcStatus *status.Status) (codes.Code, string) {
	switch rpcStatus.Code() {
	case grpc_codes.Unknown,
		grpc_codes.DeadlineExceeded,
		grpc_codes.Unimplemented,
		grpc_codes.Internal,
		grpc_codes.Unavailable,
		grpc_codes.DataLoss:
		return codes.Error, rpcStatus.Message()
	default:
		return codes.Unset, ""
	}
}
