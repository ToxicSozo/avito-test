// internal/api/generate.go
package api

//go:generate oapi-codegen -generate types  -package gen api/openapi.yaml > internal/input/http/gen/types.go
//go:generate oapi-codegen -generate chi-server,strict-server  -package gen api/openapi.yaml > internal/input/http/gen/server.go
//go:generate oapi-codegen -generate spec  -package gen api/openapi.yaml > internal/input/http/gen/spec.go

//go:generate oapi-codegen -generate types  -package client C:\Users\nebc\Desktop\avito-test\api\openapi.yml > pkg/client/http/types.go
//go:generate oapi-codegen -generate client  -package client C:\Users\nebc\Desktop\avito-test\api\openapi.yml > pkg/client/http/http_client.go
