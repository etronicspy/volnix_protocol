package app

import (
	"context"
	"fmt"
	"net"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	anteilv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/anteil/v1"
	consensusv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
	governancev1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/governance/v1"
	identv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/ident/v1"
	lizenzv1 "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/lizenz/v1"

	anteilkeeper "github.com/volnix-protocol/volnix-protocol/x/anteil/keeper"
	consensuskeeper "github.com/volnix-protocol/volnix-protocol/x/consensus/keeper"
	governancekeeper "github.com/volnix-protocol/volnix-protocol/x/governance/keeper"
	identkeeper "github.com/volnix-protocol/volnix-protocol/x/ident/keeper"
	lizenzkeeper "github.com/volnix-protocol/volnix-protocol/x/lizenz/keeper"
)

// StartGRPCServer starts a gRPC server on the given port, exposing module query services.
// It runs in a goroutine and returns when the context is cancelled.
func (app *VolnixApp) StartGRPCServer(ctx context.Context, port int) error {
	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(app.grpcUnaryInterceptor()),
	)

	// Register all module query services
	identv1.RegisterQueryServer(grpcServer, identkeeper.NewQueryServer(app.identKeeper))
	lizenzv1.RegisterQueryServer(grpcServer, lizenzkeeper.NewQueryServer(app.lizenzKeeper))
	anteilv1.RegisterQueryServer(grpcServer, anteilkeeper.NewQueryServer(app.anteilKeeper))
	consensusv1.RegisterQueryServer(grpcServer, consensuskeeper.NewQueryServer(*app.consensusKeeper))
	governancev1.RegisterQueryServer(grpcServer, governancekeeper.NewQueryServer(app.governanceKeeper))

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	go func() {
		_ = grpcServer.Serve(lis)
		// Serve returns when GracefulStop is called; ignore error on shutdown
	}()

	return nil
}

// grpcUnaryInterceptor injects sdk.Context into the request context for query handlers.
func (app *VolnixApp) grpcUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		sdkCtx := app.NewContext(true)
		ctx = sdk.WrapSDKContext(sdkCtx)
		return handler(ctx, req)
	}
}
