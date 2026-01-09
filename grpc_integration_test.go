package restclient_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmcszk/go-restclient"
	"github.com/jhump/protoreflect/desc/builder"
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

	// Create descriptors at runtime using builder
	pingReq := builder.NewMessage("PingRequest").
		AddField(builder.NewField("value", builder.FieldTypeString()).SetNumber(1))
	pingResp := builder.NewMessage("PingResponse").
		AddField(builder.NewField("value", builder.FieldTypeString()).SetNumber(1))

	pingMethod := builder.NewMethod("Ping",
		builder.RpcTypeMessage(pingReq, false),
		builder.RpcTypeMessage(pingResp, false))

	svc := builder.NewService("TestService").
		AddMethod(pingMethod)

	file := builder.NewFile("test.proto").
		SetPackageName("test").
		AddService(svc)

	fd, err := file.Build()
	if err != nil {
		return "", nil, err
	}

	// Register the file descriptor with the global registry so reflection can find it
	protoFd := fd.UnwrapFile()
	_ = protoregistry.GlobalFiles.RegisterFile(protoFd)

	// Register the service handler (dummy)
	serviceDesc := &grpc.ServiceDesc{
		ServiceName: "test.TestService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler: func(_ any, _ context.Context, _ func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
					// For testing, we just return a successful but empty response
					// if it reaches here, dispatch worked!
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
		// Test with 'dev' environment from http-client.env.json
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
		// Header should be substituted: X-Request-ID: dev-request-id
		assert.Equal(t, "dev-request-id", resp.Request.Headers.Get("X-Request-ID"))
		// Body should be substituted: "value": "hello-from-dev-env"
		// Since we don't have a real handler, we just check that the request body in resp.Request was changed
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
