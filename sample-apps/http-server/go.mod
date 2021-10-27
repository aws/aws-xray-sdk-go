module application.go

replace github.com/aws/aws-xray-sdk-go => ./aws-xray-sdk-go

require (
	github.com/aws/aws-sdk-go v1.17.12
	github.com/aws/aws-xray-sdk-go v1.3.0
	golang.org/x/net v0.0.0-20210510120150-4163338589ed
)

go 1.13
