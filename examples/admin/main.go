package main

import (
	"context"
	"log"
	"net/http"

	pb "github.com/dfanout/dfanout/proto"
)

func main() {
	ctx := context.Background()

	client := pb.NewAdminServiceJSONClient("http://localhost:8080", &http.Client{})

	// Introduce dual reads from v2 to monitor latency and error rate
	// to see if the v2 endpoint is ready for production.
	oldEndpoint := &pb.Endpoint{
		Name:    "read_likes_legacy",
		Primary: true,
		Endpoint: &pb.Endpoint_HttpEndpoint{
			HttpEndpoint: &pb.HTTPEndpoint{
				Url:    "http://localhost:8080/test", // allow url templates
				Method: "GET",
			},
		},
	}
	v2Endpoint := &pb.Endpoint{
		Name: "read_likes_v2",
		Endpoint: &pb.Endpoint_HttpEndpoint{
			HttpEndpoint: &pb.HTTPEndpoint{
				Url:    "http://localhost:8080/test2",
				Method: "GET",
				Headers: []*pb.HTTPHeader{
					{Key: "X-Extra", Values: []string{"v2"}},
				},
				TimeoutMs: 1000,
			},
		},
	}

	_, err := client.CreateFanout(ctx, &pb.CreateFanoutRequest{
		FanoutName: "read_likes",
		Endpoints:  []*pb.Endpoint{oldEndpoint, v2Endpoint},
	})
	if err != nil {
		log.Fatalf("Failed to create a fanout: %v", err)
	}

	// Remove the legacy endpoint from the read path, and update v2
	// to be primary endpoint.
	v2Endpoint.Primary = true

	_, err = client.UpdateFanout(ctx, &pb.UpdateFanoutRequest{
		FanoutName:        "read_likes",
		EndpointsToUpdate: []*pb.Endpoint{v2Endpoint},
		EndpointsToDelete: []string{oldEndpoint.Name},
	})
	if err != nil {
		log.Fatalf("Failed to update the fanout: %v", err)
	}
}
