package restclient_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmcszk/go-restclient"
	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// StartTestGRPCServer starts a minimal gRPC server with reflection enabled.
func StartTestGRPCServer() (string, func(), error) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", nil, err
	}

	s := grpc.NewServer()

	// Create descriptors at runtime using protocompile
	protoSource := `
		syntax = "proto3";
		package test;
		message PingRequest { string value = 1; }
		message PingResponse { string value = 1; }
		service TestService {
			rpc Ping(PingRequest) returns (PingResponse);
		}
	`
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: func(filename string) (io.ReadCloser, error) {
				if filename == "test.proto" {
					return io.NopCloser(strings.NewReader(protoSource)), nil
				}
				return nil, errors.New("not found")
			},
		},
	}
	fds, err := compiler.Compile(context.Background(), "test.proto")
	if err != nil {
		return "", nil, err
	}

	// Register the file descriptor with the global registry so reflection can find it
	_ = protoregistry.GlobalFiles.RegisterFile(fds[0])

	// Register the service handler (dummy)
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "test.TestService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler: func(_ any, _ context.Context, _ func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
					return nil, nil
				},
			},
		},
		Streams: []grpc.StreamDesc{},
	}

	s.RegisterService(serviceDesc, struct{}{})
	reflection.Register(s)

	go func() {
		_ = s.Serve(lis)
	}()

	stop := func() {
		s.Stop()
		_ = lis.Close()
	}

	return lis.Addr().String(), stop, nil
}

func TestGRPCE2E(t *testing.T) {
	addr, stop, err := StartTestGRPCServer()
	require.NoError(t, err)
	defer stop()

	wd, _ := os.Getwd()
	testDataDir := filepath.Join(wd, "test/data/grpc")

	t.Run("Basic Ping File", func(t *testing.T) {
		client, _ := restclient.NewClient(
			restclient.WithVars(map[string]any{
				"target": addr,
			}),
		)

		responses, err := client.ExecuteFile(context.Background(), filepath.Join(testDataDir, "ping.http"))
		require.NoError(t, err)
		require.Len(t, responses, 1)

		resp := responses[0]
		assert.NoError(t, resp.Error)
		assert.Equal(t, "OK", resp.Status)
	})

	t.Run("Variable Substitution with Env File", func(t *testing.T) {
		client, _ := restclient.NewClient(
			restclient.WithEnvironment("dev"),
			restclient.WithVars(map[string]any{
				"target": addr,
			}),
		)

		responses, err := client.ExecuteFile(context.Background(), filepath.Join(testDataDir, "vars.http"))
		require.NoError(t, err)
		require.Len(t, responses, 1)

		resp := responses[0]
		assert.NoError(t, resp.Error)
		assert.Equal(t, "dev-request-id", resp.Request.Headers.Get("X-Request-ID"))
		assert.Contains(t, resp.Request.RawBody, "hello-from-dev-env")
	})

	t.Run("Variable Substitution with Prod Env", func(t *testing.T) {
		client, _ := restclient.NewClient(
			restclient.WithEnvironment("prod"),
			restclient.WithVars(map[string]any{
				"target": addr,
			}),
		)

		responses, err := client.ExecuteFile(context.Background(), filepath.Join(testDataDir, "vars.http"))
		require.NoError(t, err)

		resp := responses[0]
		assert.Equal(t, "prod-request-id", resp.Request.Headers.Get("X-Request-ID"))
		assert.Contains(t, resp.Request.RawBody, "hello-from-prod-env")
	})
}
