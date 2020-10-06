module application.go

go 1.13

replace github.com/aws/aws-xray-sdk-go => ./aws-xray-sdk-go

require (
	github.com/aws/aws-sdk-go v1.17.12
	github.com/aws/aws-xray-sdk-go v1.0.1
	golang.org/x/net v0.0.0-20190311183353-d8887717615a
)
