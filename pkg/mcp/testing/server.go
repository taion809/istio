//  Copyright 2018 Istio Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package mcptest

import (
	"fmt"
	"io"
	"net"
	"net/url"

	mcp "istio.io/api/mcp/v1alpha1"

	"google.golang.org/grpc"

	"istio.io/istio/pkg/mcp/server"
	"istio.io/istio/pkg/mcp/snapshot"
	"istio.io/istio/pkg/mcp/testing/monitoring"
)

// Server is a simple MCP server, used for testing purposes.
type Server struct {
	// The internal snapshot.Cache that the server is using.
	Cache *snapshot.Cache

	// TypeURLs that were originally passed in.
	TypeURLs []string

	// Port that the service is listening on.
	Port int

	// The gRPC compatible address of the service.
	URL *url.URL

	gs *grpc.Server
	l  net.Listener
}

var _ io.Closer = &Server{}

// NewServer creates and starts a new MCP Server. Returns a new Server instance upon success.
// Specifying port as 0 will cause the server to bind to an arbitrary port. This port can be queried
// from the Port field of the returned server struct.
func NewServer(port int, typeUrls []string) (*Server, error) {
	cache := snapshot.New(snapshot.DefaultGroupIndex)
	s := server.New(cache, typeUrls, server.NewAllowAllChecker(), mcptestmon.NewInMemoryServerStatsContext())

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	p := l.Addr().(*net.TCPAddr).Port

	u, err := url.Parse(fmt.Sprintf("mcp://localhost:%d", p))
	if err != nil {
		_ = l.Close()
		return nil, err
	}

	gs := grpc.NewServer()

	mcp.RegisterAggregatedMeshConfigServiceServer(gs, s)
	go func() { _ = gs.Serve(l) }()

	return &Server{
		Cache:    cache,
		TypeURLs: typeUrls,
		Port:     p,
		URL:      u,
		gs:       gs,
		l:        l,
	}, nil
}

// Close implement io.Closer.Close
func (t *Server) Close() (err error) {
	if t.gs != nil {
		t.gs.GracefulStop()
		t.gs = nil
	}

	t.l = nil // gRPC stack will close this
	t.Cache = nil
	t.TypeURLs = nil
	t.Port = 0

	return
}
