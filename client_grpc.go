package restclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jhump/protoreflect/v2/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
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

func resolveGRPCMethod(ctx context.Context, conn *grpc.ClientConn, path string) (protoreflect.MethodDescriptor, error) {
	refClient := grpcreflect.NewClientAuto(ctx, conn)
	defer refClient.Reset()

	serviceName, methodName := splitGRPCMethod(path)
	if serviceName == "" || methodName == "" {
		return nil, fmt.Errorf("invalid gRPC method path: %s. Expected /package.Service/Method", path)
	}

	fd, err := refClient.FileContainingSymbol(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve service %s: %w", serviceName, err)
	}

	// Find the service in the file descriptor
	lastDot := strings.LastIndex(serviceName, ".")
	shortServiceName := serviceName
	if lastDot != -1 {
		shortServiceName = serviceName[lastDot+1:]
	}

	sd := fd.Services().ByName(protoreflect.Name(shortServiceName))
	if sd == nil {
		return nil, fmt.Errorf("service %s not found in descriptors", serviceName)
	}

	mdDesc := sd.Methods().ByName(protoreflect.Name(methodName))
	if mdDesc == nil {
		return nil, fmt.Errorf("method %s not found in service %s", methodName, serviceName)
	}
	return mdDesc, nil
}

func prepareGRPCRequest(mdDesc protoreflect.MethodDescriptor, rawBody string) (*dynamicpb.Message, error) {
	reqMsg := dynamicpb.NewMessage(mdDesc.Input())
	if rawBody != "" {
		if err := protojson.Unmarshal([]byte(rawBody), reqMsg); err != nil {
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
	reqMsg *dynamicpb.Message,
	mdDesc protoreflect.MethodDescriptor,
) {
	respMsg := dynamicpb.NewMessage(mdDesc.Output())
	startTime := time.Now()

	// gRPC full method path is /package.Service/MethodName
	fullMethodPath := fmt.Sprintf("/%s/%s", mdDesc.Parent().FullName(), mdDesc.Name())

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

func (p *grpcProcessor) handleSuccess(respMsg *dynamicpb.Message) {
	bodyBytes, err := protojson.Marshal(respMsg)
	if err != nil {
		p.clientResponse.Error = fmt.Errorf("failed to marshal response to JSON: %w", err)
	} else {
		p.clientResponse.Body = bodyBytes
		p.clientResponse.BodyString = string(bodyBytes)
	}
}

// splitGRPCMethod splits a full gRPC method path into service and method names.
func splitGRPCMethod(fullMethod string) (serviceName, methodName string) {
	fullMethod = strings.TrimPrefix(fullMethod, "/")
	parts := strings.Split(fullMethod, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
