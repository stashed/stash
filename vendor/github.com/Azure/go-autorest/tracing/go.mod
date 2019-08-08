module github.com/Azure/go-autorest/tracing

go 1.12

require (
	// use older versions to avoid taking a dependency on protobuf v1.3+
	contrib.go.opencensus.io/exporter/ocagent v0.4.6
	github.com/grpc-ecosystem/grpc-gateway v1.9.5 // indirect
	go.opencensus.io v0.19.2
)
