module github.com/supercargo-dev/supercargo-sdk-go

go 1.25.5

require (
	github.com/stretchr/testify v1.11.1
	github.com/supercargo-dev/core/gen v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.82.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/supercargo-dev/core/gen => ../core/gen
