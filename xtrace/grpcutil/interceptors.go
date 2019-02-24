package grpcutil

import (
	"fmt"
	xtr "github.com/brown-csci1380/tracing-framework-go/xtrace/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
)

// Handles propagation of x-trace metadata around grpc server requests (as the ServerOption to grpc.NewServer)
var XTraceServerInterceptor grpc.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		fmt.Fprintln(os.Stderr, "no metadata in request context.")
	}

	GRPCRecieved(md, fmt.Sprintf("Recieved %s, args: %s", info.FullMethod, req))
	resp, err := handler(ctx, req)
	if err != nil {
		xtr.Logf("Returning from %s, error: %s", info.FullMethod, err.Error())
	} else {
		xtr.Logf("Returning from %s, response: %s", info.FullMethod, resp)
	}
	grpc.SetHeader(ctx, metadata.Pairs(GRPCMetadata()...))
	return resp, err
}

// Handles propagation of x-trace metadata around grpc server stream RPCs (as a ServerOption to grpc.NewServer)
var XTraceStreamServerInterceptor grpc.StreamServerInterceptor = func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		fmt.Fprintln(os.Stderr, "no metadata in request context.")
	}

	GRPCRecieved(md, fmt.Sprintf("Recieved %s", info.FullMethod))
	err := handler(srv, ss)
	if err != nil {
		xtr.Logf("Failed to create remote stream for %s, error: %s", info.FullMethod, err.Error())
	} else {
		xtr.Logf("Cread remote stream for %s, successful", info.FullMethod)
	}
	ss.SetHeader(metadata.Pairs(GRPCMetadata()...))
	return err
}

// Handles propagation of x-trace metadata around grpc remote calls (as the argument to grpc.WithUnaryInterceptor)
var XTraceClientInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	xtr.Logf("Calling %s, arg: %v", method, req)
	var md metadata.MD
	err := invoker(metadata.NewOutgoingContext(ctx, metadata.Pairs(GRPCMetadata()...)), method, req, reply, cc, append(opts, grpc.Header(&md))...)
	GRPCReturned(md, fmt.Sprintf("Returned from remote %s, error: %v, value: %v", method, err, reply))
	return err
}

// Handles propagation of x-trace metadata around grpc stream calls (as the argument to grpc.WithStreamInterceptor)
var XTraceStreamClientInterceptor grpc.StreamClientInterceptor = func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	xtr.Logf("Calling %s, desc: %v", method, desc)
	var md metadata.MD
	cs, err := streamer(metadata.NewOutgoingContext(ctx, metadata.Pairs(GRPCMetadata()...)), desc, cc, method, append(opts, grpc.Header(&md))...)
	GRPCReturned(md, fmt.Sprintf("Recieved remote stream for %v: error: %v, stream: %v", method, err, cs))
	return cs, err
}
