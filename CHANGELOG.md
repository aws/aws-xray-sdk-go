Release v1.0.0-rc.12 (2019-06-11)
================================
### SDK Breaking Changes
* Updates `sampling.NewProxy` method to be private. [PR #93](https://github.com/aws/aws-xray-sdk-go/pull/93)

### SDK Enhancements
* Fixes a bug for failing to close in-progress `connect` subsegments in some cases. [PR #102](https://github.com/aws/aws-xray-sdk-go/pull/102)
* Fixes data races condition. [PR #103](https://github.com/aws/aws-xray-sdk-go/pull/103)
* Fixes a nil pointer issue. [PR #109](https://github.com/aws/aws-xray-sdk-go/pull/109)
* Refactors `newGlobalConfig` to avoid initializing log. [PR #96](https://github.com/aws/aws-xray-sdk-go/pull/96)
* Adds `-race` to travis test script. [PR #104](https://github.com/aws/aws-xray-sdk-go/pull/104)
* Fixes data race condition for parallel http client request. [PR #100](https://github.com/aws/aws-xray-sdk-go/pull/100)
* Adds support for `tx.Prepare`. [PR #95](https://github.com/aws/aws-xray-sdk-go/pull/95)
* Fixes race bugs with `ClientTrace`. [PR #115](https://github.com/aws/aws-xray-sdk-go/pull/115)
* Updates lock abstraction for `defaultLogger`. [PR #113](https://github.com/aws/aws-xray-sdk-go/pull/113)
* Adds `golangci-lint` into travis CI. [PR #114](https://github.com/aws/aws-xray-sdk-go/pull/114)
* Fixes uncaught error on SQL url parse. [PR #121](https://github.com/aws/aws-xray-sdk-go/pull/121)

Release v1.0.0-rc.11 (2019-03-15)
================================
### SDK Breaking Changes
* Dropped support for go versions 1.7 and 1.8. Users will need to use go version 1.9 or higher. [PR #91](https://github.com/aws/aws-xray-sdk-go/pull/91)

### SDK Enhancements
* Adds support for go [dep](https://github.com/golang/dep) and go modules. [PR #90](https://github.com/aws/aws-xray-sdk-go/pull/90)
* Fixes a bug where optional interfaces on `http.ResponseWriter` (e.g. `http.Flusher`, `http.CloseNotifier`, etc.)
were not visible due to how the xray Handler wrapped the `http.ResponseWriter`. [PR #91](https://github.com/aws/aws-xray-sdk-go/pull/91)

Release v1.0.0-rc.10 (2019-02-19)
================================
### SDK Breaking Changes
* `xray.Config{}` fields `LogLevel` and `LogFormat` are deprecated and no longer have any effect. Users will have to reset their min log level if they weren't using the default of "info" using `xray.SetLogger()` . The log levels `Trace` and `Tracef` are replaced by `Debug` and `Debugf` respectively. [PR #82](https://github.com/aws/aws-xray-sdk-go/pull/82), [Issue #15](https://github.com/aws/aws-xray-sdk-go/issues/15)

### SDK Enhancements
* Don't try to udp dial emitter at package load time [PR #83](https://github.com/aws/aws-xray-sdk-go/pull/83)
* Explicit plugin initialization [PR #81](https://github.com/aws/aws-xray-sdk-go/pull/81)

Release v1.0.0-rc.9 (2018-12-20)
================================
### SDK Enhancements
* Fix http2 datarace in unit test: [PR #72](https://github.com/aws/aws-xray-sdk-go/pull/72)
* Support passing customized emitter: [PR #76](https://github.com/aws/aws-xray-sdk-go/pull/76)
* Apply Context Missing Strategy if segment is nil in SQL
* Remove error message content check for certain daemon config unit tests

Release v1.0.0-rc.8 (2018-10-04)
================================
### SDK Bugs
* Adding hostname support for daemon address parsing

### SDK Enhancements
* Increase unit test coverage

Release v1.0.0-rc.7 (2018-09-27)
================================
### SDK Breaking changes
* `samplingRule` is an exported type : PR[#67](https://github.com/aws/aws-xray-sdk-go/pull/67)
* Renamed `SamplingRule` structure to `Properties`

Release v1.0.0-rc.6 (2018-09-25)
================================
### SDK Breaking changes
* The default sampling strategy is `CentralizedStrategy` that launches background tasks to poll sampling rules from X-Ray backend. See the new default sampling strategy in more details 
here: [Link](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-go-configuration.html#xray-sdk-go-configuration-sampling)
* The `ShouldTrace()` function in the `Strategy` interface now takes a `Request` structure for sampling rule matching and returns `Decision` object
* Updated `aws-sdk-go` version in `glide.yaml` file to `1.15.23`.
* Modified `Rule` structure : It contains `samplingRule` nested structure

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

