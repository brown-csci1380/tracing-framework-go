package grpcutil

import (
	"fmt"
	xtr "github.com/crotger/tracing-framework-go/xtrace/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"os"
)

// Handles propagation of x-trace metadata around grpc server requests (as the ServerOption to grpc.NewServer)
var XTraceServerInterceptor grpc.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromContext(ctx)
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

// Handles propagation of x-trace metadata around grpc remote calls (as the DialOption to grpc.WithUnaryInterceptor)
var XTraceClientInterceptor grpc.UnaryClientInterceptor = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	xtr.Logf("Calling %s, arg: %v", method, req)
	var md metadata.MD
	err := invoker(metadata.NewContext(ctx, metadata.Pairs(GRPCMetadata()...)), method, req, reply, cc, append(opts, grpc.Header(&md))...)
	GRPCReturned(md, fmt.Sprintf("Returned from remote %s, error: %s, value: %v", method, err.Error(), reply))
	return err
}
