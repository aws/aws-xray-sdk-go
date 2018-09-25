Release v1.0.0-rc.6 (2018-09-25)
================================
### SDK Breaking changes
* The default sampling strategy is `CentralizedStrategy` that launches background tasks to poll sampling rules from X-Ray backend. See the new default sampling strategy in more details 
here: [Link](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-go-configuration.html#xray-sdk-go-configuration-sampling)
* The `ShouldTrace()` function in the `Strategy` interface now takes a `Request` structure for sampling rule matching and returns `Decision` object
* Updated `aws-sdk-go` version in `glide.yaml` file to `1.15.23`.

### SDK Enhancements
* Environment variable `AWS_XRAY_DAEMON_ADDRESS` now takes an additional notation in `tcp:127.0.0.1:2000 udp:127.0.0.2:2001` to set TCP and UDP destination separately. By default it assumes a X-Ray daemon listening to both UDP and TCP traffic on 127.0.0.1:2000.
* Update DefaultSamplingRules.json file. i.e. service_name has been replaced to host and version changed to 2. SDK still supports v1 JSON file.
* Fix httptrace datarace : [PR #62](https://github.com/aws/aws-xray-sdk-go/pull/62)
* Fix sub-segment datarace : [PR #61](https://github.com/aws/aws-xray-sdk-go/pull/61)

Release v1.0.0-rc.5 (2018-05-15)
================================
### SDK Bugs
* Traced aws client allows requests to proceed without segments presenting.

### SDK Enhancements
* Increase unit test coverage.
* Update `aws-sdk-go` version in `glide.yaml` file.

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

