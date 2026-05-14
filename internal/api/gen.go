// Package api exposes the HTTP API surface. The types and Gin server
// interface in `openapi.gen.go` are generated from `api/openapi.yaml`;
// do not edit them by hand. Hand-written code (handler implementations,
// middleware) lives in sibling files in this package.
package api

// `go generate` runs in this file's directory; cd to the repo root so the
// relative paths inside codegen.yaml resolve correctly.
//go:generate sh -c "cd ../.. && go tool oapi-codegen -config codegen.yaml api/openapi.yaml"
