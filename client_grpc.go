package restclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	//nolint:staticcheck // usage of deprecated jhump/protoreflect is intentional
	"github.com/jhump/protoreflect/desc"
	//nolint:staticcheck // usage of deprecated jhump/protoreflect is intentional
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
func (*Client) executeGRPCRequest(ctx context.Context, rcRequest *Request) (*Response, error) {
	if rcRequest == nil {
		return nil, errors.New("cannot execute a nil request")
	}

	clientResponse := &Response{Request: rcRequest, Proto: "gRPC", Headers: make(http.Header)}

	// 1. Prepare Metadata
	ctx = prepareGRPCMetadata(ctx, rcRequest.Headers)

	// 2. Connect
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(rcRequest.URL.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		clientResponse.Error = fmt.Errorf("failed to create gRPC client: %w", err)
		return clientResponse, nil
	}
	defer func() { _ = conn.Close() }()

	// 3. Resolve Service and Method
	mdDesc, err := resolveGRPCMethod(dialCtx, conn, rcRequest.URL.Path)
	if err != nil {
		clientResponse.Error = err
		return clientResponse, nil
	}

	// 4. Prepare and Invoke
	reqMsg, err := prepareGRPCRequest(mdDesc, rcRequest.RawBody)
	if err != nil {
		clientResponse.Error = err
		return clientResponse, nil
	}

	p := &grpcProcessor{conn: conn, clientResponse: clientResponse}
	p.invokeAndProcess(ctx, reqMsg, mdDesc)

	return clientResponse, nil
}

func resolveGRPCMethod(ctx context.Context, conn *grpc.ClientConn, path string) (*desc.MethodDescriptor, error) {
	refClient := grpcreflect.NewClientV1(ctx, reflectpb.NewServerReflectionClient(conn))
	defer refClient.Reset()

	serviceName, methodName := splitGRPCMethod(path)
	if serviceName == "" || methodName == "" {
		return nil, fmt.Errorf("invalid gRPC method path: %s. Expected /package.Service/Method", path)
	}

	sd, err := refClient.ResolveService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve service %s: %w", serviceName, err)
	}

	mdDesc := sd.FindMethodByName(methodName)
	if mdDesc == nil {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}
	return mdDesc, nil
}

func prepareGRPCRequest(mdDesc *desc.MethodDescriptor, rawBody string) (*dynamic.Message, error) {
	reqMsg := dynamic.NewMessage(mdDesc.GetInputType())
	if rawBody != "" {
		if err := reqMsg.UnmarshalJSON([]byte(rawBody)); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request body to protobuf: %w", err)
		}
	}
	return reqMsg, nil
}

type grpcProcessor struct {
	conn           *grpc.ClientConn
	clientResponse *Response
}

func prepareGRPCMetadata(ctx context.Context, headers http.Header) context.Context {
	md := metadata.New(make(map[string]string))
	for key, values := range headers {
		for _, value := range values {
			md.Append(key, value)
		}
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func (p *grpcProcessor) invokeAndProcess(
	ctx context.Context,
	reqMsg *dynamic.Message,
	mdDesc *desc.MethodDescriptor,
) {
	respMsg := dynamic.NewMessage(mdDesc.GetOutputType())
	startTime := time.Now()
	fullMethodPath := fmt.Sprintf("/%s/%s", mdDesc.GetService().GetFullyQualifiedName(), mdDesc.GetName())

	var header, trailer metadata.MD
	err := p.conn.Invoke(ctx, fullMethodPath, reqMsg, respMsg, grpc.Header(&header), grpc.Trailer(&trailer))
	p.clientResponse.Duration = time.Since(startTime)

	p.processMetadata(header, trailer)

	if err != nil {
		st, _ := status.FromError(err)
		p.clientResponse.StatusCode = int(st.Code())
		p.clientResponse.Status = st.Code().String()
		p.clientResponse.Error = err
	} else {
		p.clientResponse.StatusCode = int(codes.OK)
		p.clientResponse.Status = codes.OK.String()
		p.handleSuccess(respMsg)
	}
}

func (p *grpcProcessor) processMetadata(header, trailer metadata.MD) {
	for k, v := range header {
		for _, val := range v {
			p.clientResponse.Headers.Add(k, val)
		}
	}
	for k, v := range trailer {
		for _, val := range v {
			p.clientResponse.Headers.Add(k, val)
		}
	}
}

func (p *grpcProcessor) handleSuccess(respMsg *dynamic.Message) {
	bodyBytes, err := respMsg.MarshalJSON()
	if err != nil {
		p.clientResponse.Error = fmt.Errorf("failed to marshal response to JSON: %w", err)
	} else {
		p.clientResponse.Body = bodyBytes
		p.clientResponse.BodyString = string(bodyBytes)
	}
}

// splitGRPCMethod splits a full gRPC method path into service and method names.
// e.g. "/package.Service/MethodName" -> ("package.Service", "MethodName")
func splitGRPCMethod(fullMethod string) (serviceName, methodName string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
