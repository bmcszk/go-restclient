package restclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
)

// executeGRPCRequest executes a gRPC request and returns the Response.
func (c *Client) executeGRPCRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		return nil, fmt.Errorf("cannot execute a nil request")
	}

	clientResponse := &Response{Request: rcRequest, Proto: "gRPC", Headers: make(http.Header)}

	// 1. Prepare Target and Method
	// URL.Host should be the target, URL.Path should be /package.Service/Method
	target := rcRequest.URL.Host
	method := rcRequest.URL.Path

	// 2. Build Metadata (Headers)
	md := metadata.New(make(map[string]string))
	for key, values := range rcRequest.Headers {
		for _, value := range values {
			md.Append(key, value)
		}
	}
	ctx = metadata.NewOutgoingContext(ctx, md)

	// 3. Connect (MVP starts with insecure)
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // default connection timeout
	defer cancel()

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		clientResponse.Error = fmt.Errorf("failed to create gRPC client: %w", err)
		return clientResponse, nil
	}
	defer conn.Close()

	// 4. Invocation logic using Server Reflection
	refClient := grpcreflect.NewClientV1(dialCtx, reflectpb.NewServerReflectionClient(conn))
	defer refClient.Reset()

	serviceName, methodName := splitGRPCMethod(method)
	if serviceName == "" || methodName == "" {
		clientResponse.Error = fmt.Errorf("invalid gRPC method path: %s. Expected /package.Service/Method", method)
		return clientResponse, nil
	}

	sd, err := refClient.ResolveService(serviceName)
	if err != nil {
		clientResponse.Error = fmt.Errorf("failed to resolve service %s: %w", serviceName, err)
		return clientResponse, nil
	}

	mdDesc := sd.FindMethodByName(methodName)
	if mdDesc == nil {
		clientResponse.Error = fmt.Errorf("method %s not found in service %s", methodName, serviceName)
		return clientResponse, nil
	}

	// Prepare request message
	reqMsg := dynamic.NewMessage(mdDesc.GetInputType())

	if rcRequest.RawBody != "" {
		if err := reqMsg.UnmarshalJSON([]byte(rcRequest.RawBody)); err != nil {
			clientResponse.Error = fmt.Errorf("failed to unmarshal request body to protobuf: %w", err)
			return clientResponse, nil
		}
	}

	// Execute call
	respMsg := dynamic.NewMessage(mdDesc.GetOutputType())

	startTime := time.Now()
	fullMethodPath := fmt.Sprintf("/%s/%s", serviceName, methodName)

	var header, trailer metadata.MD
	err = conn.Invoke(ctx, fullMethodPath, reqMsg, respMsg, grpc.Header(&header), grpc.Trailer(&trailer))
	clientResponse.Duration = time.Since(startTime)

	// Process response metadata
	for k, v := range header {
		for _, val := range v {
			clientResponse.Headers.Add(k, val)
		}
	}
	for k, v := range trailer {
		for _, val := range v {
			clientResponse.Headers.Add(k, val)
		}
	}

	if err != nil {
		st, _ := status.FromError(err)
		clientResponse.StatusCode = int(st.Code())
		clientResponse.Status = st.Code().String()
		clientResponse.Error = err
	} else {
		clientResponse.StatusCode = int(codes.OK)
		clientResponse.Status = codes.OK.String()

		bodyBytes, err := respMsg.MarshalJSON()
		if err != nil {
			clientResponse.Error = fmt.Errorf("failed to marshal response to JSON: %w", err)
		} else {
			clientResponse.Body = bodyBytes
			clientResponse.BodyString = string(bodyBytes)
		}
	}

	return clientResponse, nil
}

// splitGRPCMethod splits a full gRPC method path into service and method names.
// e.g. "/package.Service/MethodName" -> ("package.Service", "MethodName")
func splitGRPCMethod(fullMethod string) (string, string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
