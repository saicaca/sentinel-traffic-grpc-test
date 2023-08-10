/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"github.com/alibaba/sentinel-golang/core/route"
	"github.com/alibaba/sentinel-golang/core/route/base"
	"log"
	"time"

	adapter "github.com/alibaba/sentinel-golang/pkg/adapters/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:50051", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

func init() {
	trafficRouter := base.TrafficRouter{
		Host: []string{"test-provider"},
		Http: []*base.HTTPRoute{
			{
				Name: "test-traffic-provider-rule",
				Match: []*base.HTTPMatchRequest{
					{
						Headers: map[string]*base.StringMatch{
							"x-tag": {Exact: "v2"},
						},
						Method: &base.StringMatch{
							Exact: "hello",
						},
					},
				},
				Route: []*base.HTTPRouteDestination{
					{
						Weight: 1,
						Destination: &base.Destination{
							Host:   "test-provider",
							Subset: "v3",
						},
					},
				},
			},
		},
	}

	virtualWorkload := base.VirtualWorkload{
		Host: "test-provider",
		Subsets: []*base.Subset{
			{
				Name: "v1",
				Labels: map[string]string{
					"instance-tag": "v1",
				},
			}, {
				Name: "v2",
				Labels: map[string]string{
					"instance-tag": "v2",
				},
			}, {
				Name: "v3",
				Labels: map[string]string{
					"instance-tag": "v3",
				},
			},
		},
	}

	route.SetAppName("test-consumer")
	route.SetTrafficRouterList([]*base.TrafficRouter{&trafficRouter})
	route.SetVirtualWorkloadList([]*base.VirtualWorkload{&virtualWorkload})
}

func main() {
	cm := &route.ClusterManager{}
	cm.InstanceManager = &route.BasicInstanceManager{}

	instanceList := []*base.Instance{
		{
			AppName: "test-provider",
			Host:    "127.0.0.1",
			Port:    50051,
			Metadata: map[string]string{
				"instance-tag": "v1",
			},
		}, {
			AppName: "test-provider",
			Host:    "127.0.0.1",
			Port:    50052,
			Metadata: map[string]string{
				"instance-tag": "v2",
			},
		}, {
			AppName: "test-provider",
			Host:    "127.0.0.1",
			Port:    50053,
			Metadata: map[string]string{
				"instance-tag": "v3",
			},
		},
	}

	cm.InstanceManager.StoreInstances(instanceList)
	cm.RouterFilterList = []route.RouterFilter{
		&route.BasicRouterFilter{},
	}

	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.Dial(
		"test-provider/hello",
		grpc.WithContextDialer(adapter.NewDialer(
			adapter.WithClusterManager(cm),
			adapter.WithHeaders(map[string]string{
				"x-tag": "v2",
			}),
		)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
