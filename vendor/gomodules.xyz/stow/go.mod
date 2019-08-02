module gomodules.xyz/stow

require (
	cloud.google.com/go v0.39.0 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.4.12 // indirect
	github.com/Azure/azure-sdk-for-go v31.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.5.0 // indirect
	github.com/Azure/go-autorest/autorest/mocks v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.2.0 // indirect
	github.com/Azure/go-autorest/tracing v0.2.0 // indirect
	github.com/aws/aws-sdk-go v1.20.20
	github.com/cheekybits/is v0.0.0-20150225183255-68e9c0620927
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dnaeon/go-vcr v0.0.0-20180814043457-aafff18a5cc2 // indirect
	github.com/google/readahead v0.0.0-20161222183148-eaceba169032 // indirect
	github.com/ncw/swift v1.0.47
	github.com/pkg/errors v0.8.1
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/stretchr/testify v1.3.0 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.7.0
	gopkg.in/kothar/go-backblaze.v0 v0.0.0-20190520213052-702d4e7eb465
	gopkg.in/yaml.v2 v2.2.1 // indirect
)

replace (
	contrib.go.opencensus.io/exporter/ocagent => contrib.go.opencensus.io/exporter/ocagent v0.3.0
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.3.0+incompatible
	github.com/census-instrumentation/opencensus-proto => github.com/census-instrumentation/opencensus-proto v0.1.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.2.0
	go.opencensus.io => go.opencensus.io v0.21.0
)
