// Package client provides gRPC and REST clients for interacting with the Tharsis API.
//
// There are two client types, each serving a different transport:
//
//   - [GRPCClient] handles all standard API operations (workspaces, runs, modules, etc.)
//     over gRPC. It auto-discovers the gRPC endpoint via the Tharsis service discovery
//     document and supports TLS, retry policies, and keepalive.
//
//   - [RESTClient] handles binary upload/download operations (configuration versions,
//     module packages, provider binaries, mirror packages) over HTTP PUT/GET against
//     the Terraform-compatible REST API.
//
// Both clients accept a [TokenResolver] for authentication. See the [token] subpackage
// for built-in resolver implementations and environment-based configuration.
//
// # Usage
//
//	grpcClient, err := client.NewGRPCClient(ctx, &client.GRPCClientConfig{
//	    HTTPEndpoint:  endpoint,
//	    TokenResolver: resolver,
//	})
//	restClient, err := client.NewRESTClient(&client.RESTClientConfig{
//	    Endpoint:      endpoint,
//	    TokenResolver: resolver,
//	})
package client
