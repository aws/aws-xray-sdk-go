module application.go

replace github.com/aws/aws-xray-sdk-go => ./aws-xray-sdk-go

require (
	github.com/aws/aws-sdk-go v1.17.12
	github.com/aws/aws-xray-sdk-go v1.6.0
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.34.0 // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.35.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)

go 1.17
