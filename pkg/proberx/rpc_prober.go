package proberx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"integrated-exporter/config"
	"integrated-exporter/pkg/constantx"
	"log"
	"strings"
	"time"
)

func ProbeRpc(rs config.RpcService) error {
	timeout, err := time.ParseDuration(rs.Timeout)
	if err != nil {
		log.Printf("Failed to parse timeout duration for probe %s %v: %v", constantx.RpcService, rs.Name, err)
		return err
	}
	client, err := grpc.NewClient(rs.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to connect to server for %s %v: %v", constantx.RpcService, rs.Name, err)
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	md := metadata.New(nil)
	md.Append("Authorization", rs.Token)
	ctx = metadata.NewOutgoingContext(ctx, md)
	refClient := grpcreflect.NewClientAuto(context.Background(), client)
	refClient.AllowMissingFileDescriptors()
	descriptorSource := grpcurl.DescriptorSourceFromServer(context.Background(), refClient)
	auth := fmt.Sprintf("Authorization: %s", rs.Token)
	resolver := grpcurl.AnyResolverFromDescriptorSource(descriptorSource)
	var resp bytes.Buffer
	handler := grpcurl.DefaultEventHandler{
		Out:       &resp,
		Formatter: grpcurl.NewJSONFormatter(false, resolver),
	}
	next := grpcurl.NewJSONRequestParser(bytes.NewBuffer([]byte(rs.Body)), resolver).Next
	err = grpcurl.InvokeRPC(ctx, descriptorSource, client, rs.RpcMethod, []string{auth}, &handler, next)
	if err != nil {
		log.Printf("Failed to invoke method for %v %v: %v", constantx.RpcService, rs.Name, err)
		return err
	}

	if rs.Response != "" {
		if !strings.Contains(resp.String(), rs.Response) {
			return errors.New(fmt.Sprintf("%s %s probe response does not contain %s", constantx.RpcService, rs.Name, rs.Response))
		}
	}
	return nil
}