Release v1.0.0-rc.4 (2018-04-13)
================================
### SDK Bugs
* Flush subsegments generated inside Lambda function.

### SDK Enhancements
* Capture extra request id with header `x-amz-id-2` for AWS S3 API calls.

Release v1.0.0-rc.3 (2018-04-04)
================================
### SDK Enhancements
* Fix data race condition in BeginSubsegment and Close methods.
* Remove port number in client_ip when X-Forwarded_For is empty.
* Fix version issue for httptrace library.

Release v1.0.0-rc.2 (2018-03-22)
================================
### SDK Enhancements
* Assign subsegment a defined name when HTTP interceptor creates an invalid subsegment due to URL is not available.
* Fetch ContextMissingStrategy from segment if its subsegment is nil.

Release v1.0.0-rc.1 (2018-01-15)
================================
### SDK Feature Updates
* Support for tracing within AWS Lambda functions.
* Support method to inject configuration setting in `Context`.
* Support send subsegments method.

### SDK Enhancements
* Set origin value for each plugin and also show service name in segment document plugin section.
* Remove attempt number when AWS SDK retries.
* Make HTTP requests if segment doesn't exist.
* Add Go SDK Version, Go Compiler and X-Ray SDK information in segment document.
* Set remote value when AWS request fails due to a service-side error. 

Release v0.9.4 (2017-09-08)
===========================
### SDK Enhancements
* Refactor code to fit Go Coding Standard.
* Update README.
* `aws-xray-sdk-go/xray`: make HTTP request if segment cannot be found.

