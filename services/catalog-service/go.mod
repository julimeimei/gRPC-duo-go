module github.com/julimeimei/grpc-duo-go/services/catalog-service

go 1.26.0

toolchain go1.26.9

require (
	github.com/julimeimei/grpc-duo-go/gen/go v0.0.0
	google.golang.org/grpc v1.83.2
)

require (
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/julimeimei/grpc-duo-go/gen/go => ../../gen/go
