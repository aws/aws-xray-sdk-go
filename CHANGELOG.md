Unreleased
===============================
### SDK Breaking Changes

### SDK Enhancements

### SDK Bugs

Release v2.0.0 (2025-02-19)
===============================
### SDK Breaking Changes
* Remove v1 of aws-sdk-go and update sampler to use http client instead of aws sdk [#PR 485](https://github.com/aws/aws-xray-sdk-go/pull/485)
  * Addresses CVE-2020-8912 and CVE-2020-8911

### SDK Enhancements

### SDK Bugs
* Update golang.org/x/net to v0.33.0 [#PR 487](https://github.com/aws/aws-xray-sdk-go/pull/487)
  * Addresses [CVE-2024-45338](https://github.com/aws/aws-xray-sdk-go/security/dependabot/67)

Release v1.8.5 (2024-11-13)
================================
### SDK Enhancements

### SDK Bugs
*  Fix and upgrade protobuf  [#PR 469](https://github.com/aws/aws-xray-sdk-go/pull/469)
*  Update fasthttp to v1.52.0 for RSA support [#PR 478](https://github.com/aws/aws-xray-sdk-go/pull/478)


Release v1.8.4 (2024-04-24)
================================
### SDK Enhancements

### SDK Bugs
*  Updated go-sqlmock to v1.5.1 [#PR 451](https://github.com/aws/aws-xray-sdk-go/pull/451)
*  Fix HTTP2 rapid reset vulnerability [#PR 428](https://github.com/aws/aws-xray-sdk-go/pull/428)
*  Omit URL's password when stringifying URL for segment name [#PR 422](https://github.com/aws/aws-xray-sdk-go/pull/422)


Release v1.8.3 (2023-11-13)
================================
### SDK Enhancements

### SDK Bugs
* Update AWS SDK for Go depdencies [#PR 430](https://github.com/aws/aws-xray-sdk-go/pull/430)
* Fix fatal error: concurrent map iteration and map write [#PR 457](https://github.com/aws/aws-xray-sdk-go/pull/457)
* Bump google.golang.org/protobuf from 1.31.0 to 1.33.0 [#PR 459](https://github.com/aws/aws-xray-sdk-go/pull/459)
* Bump golang.org/x/net from 0.18.0 to 0.23.0 [#PR 464](https://github.com/aws/aws-xray-sdk-go/pull/464)


Release v1.8.2 (2023-09-28)
================================
### SDK Enhancements
* Change how SDK sets the context for AWS SDK calls [#PR 418](https://github.com/aws/aws-xray-sdk-go/pull/418)

### SDK Bugs
*  Suppress Panic in Emitter [#PR 419](https://github.com/aws/aws-xray-sdk-go/pull/419)


Release v1.8.1 (2023-02-27)
================================
### SDK Enhancements

### SDK Bugs
* Fix Sample App Solution Stack and GO version [#PR 388](https://github.com/aws/aws-xray-sdk-go/pull/388)
* Fix AWS GO SDK Vulnerability [#PR 390](https://github.com/aws/aws-xray-sdk-go/pull/390)
* Fix mutex deadlock [#PR 393](https://github.com/aws/aws-xray-sdk-go/pull/393)
* Update golang.org/x/text to fix CVEs [#PR 400](https://github.com/aws/aws-xray-sdk-go/pull/400)
* Fix CVEs in integration tests [#PR 401](https://github.com/aws/aws-xray-sdk-go/pull/401)
* Update sample app dependencies [#PR 405](https://github.com/aws/aws-xray-sdk-go/pull/405)
* Update golang.org/x/net to fix CVEs [#PR 406](https://github.com/aws/aws-xray-sdk-go/pull/406)


Release v1.8.0 (2022-11-08)
================================
### SDK Enhancements
* Oversampling Mitigation [#PR 381](https://github.com/aws/aws-xray-sdk-go/pull/381)
* Changed Missing Context default strategy to log [#PR 382](https://github.com/aws/aws-xray-sdk-go/pull/382)

### SDK Bugs


Release v1.7.1 (2022-09-14)
================================
### SDK Enhancements

### SDK Bugs
* Replace error type assertions with errors.As in DefaultFormattingStrategy [#PR 353](https://github.com/aws/aws-xray-sdk-go/pull/353)
* Dummy segments don't need cancel go routine [#PR 365](https://github.com/aws/aws-xray-sdk-go/pull/365)
* Strip X-Amz-Security-Token from SQL URIs [#PR 367](https://github.com/aws/aws-xray-sdk-go/pull/367)
* Upgrading Go Version [#PR 379](https://github.com/aws/aws-xray-sdk-go/pull/379)


Release v1.7.0 (2022-04-11)
================================
### SDK Enhancements
* Removes deprecated method checks in SQL instrumentation [PR #341](https://github.com/aws/aws-xray-sdk-go/pull/341)
* Migrates private API named httpTrace public [PR #329](https://github.com/aws/aws-xray-sdk-go/pull/329)
* Migrates to use `grpc.SetHeader` API [PR #312](https://github.com/aws/aws-xray-sdk-go/pull/312)
* Removes support for go dep [PR #343](https://github.com/aws/aws-xray-sdk-go/pull/343)
* Replace error type assertions with `errors.As` [PR #353](https://github.com/aws/aws-xray-sdk-go/pull/353)

### SDK Bugs
* Fixes segment leaking issues in `BeginSegmentWithSampling` API [PR #327](https://github.com/aws/aws-xray-sdk-go/pull/327)
* Updates github.com/valyala/fasthttp dependency to v1.34.0 to fix security vulnerability issue [PR #351](https://github.com/aws/aws-xray-sdk-go/pull/351)


Release v1.6.0 (2021-07-07)
================================
### SDK Enhancements
* AWS SDK v2 instrumentation support [PR #309](https://github.com/aws/aws-xray-sdk-go/pull/309)

### SDK Bugs
* Fixed appending to existing gRPC context [PR #308](https://github.com/aws/aws-xray-sdk-go/pull/308)
* Fixed memory leak issue on fasthttp handler [PR #311](https://github.com/aws/aws-xray-sdk-go/pull/311)
* Fixed panic issue when segment is not present [PR #316](https://github.com/aws/aws-xray-sdk-go/pull/316)


Release v1.5.0 (2021-06-10)
================================
### SDK Enhancements
* gRPC instrumentation support [PR #292](https://github.com/aws/aws-xray-sdk-go/pull/292)
* fasthttp handler instrumentation support [PR #299](https://github.com/aws/aws-xray-sdk-go/pull/299)

### SDK Bugs
* Fix `AWS_XRAY_TRACING_NAME` environment variable issue when directly calling `xray.BeginSegment` API [PR #304](https://github.com/aws/aws-xray-sdk-go/pull/304)


Release v1.4.0 (2021-05-03)
================================
### SDK Enhancements
* No op trace id generation support for unsampled traces [PR #293](https://github.com/aws/aws-xray-sdk-go/pull/293)
* Refactored `httpTrace` method [PR #296](https://github.com/aws/aws-xray-sdk-go/pull/296)

### SDK Bugs
* Fix panic issue when call `AddError` function [PR #288](https://github.com/aws/aws-xray-sdk-go/pull/288)

Release v1.3.0 (2021-02-02)
================================
### SDK Enhancements
* Added SQL tracing name support (for database with same name) [PR #273](https://github.com/aws/aws-xray-sdk-go/pull/273)
* Added automated release workflow [PR #274](https://github.com/aws/aws-xray-sdk-go/pull/274)

Release v1.2.0 (2021-01-05)
================================
### SDK Enhancements
* Refresh messages are `debug` to reduce log noise. [PR #241](https://github.com/aws/aws-xray-sdk-go/pull/241)
* Added `runtime` and `runtime_version` keys [PR #245](https://github.com/aws/aws-xray-sdk-go/pull/245)
* `BeginSegment` API honors sampling rule based on `ServiceName` [PR #244](https://github.com/aws/aws-xray-sdk-go/pull/244)
* Added `IGNORE_ERROR` mode for context missing strategy [PR #253](https://github.com/aws/aws-xray-sdk-go/pull/253)
* Added X-Ray Go SDK sample apps and added Github workflow to publish sample app image tags to ECR [PR #261](https://github.com/aws/aws-xray-sdk-go/pull/261)
* Added Github workflow for end to end Integration Test for X-Ray Go SDK [PR #270](https://github.com/aws/aws-xray-sdk-go/pull/270)

### SDK Bugs
* Fix typo (Metdata -> Metadata) [PR #239](https://github.com/aws/aws-xray-sdk-go/pull/239)
* Remove Deprecated `set-env` and `add-path` syntax from workflow [PR #267](https://github.com/aws/aws-xray-sdk-go/pull/267)
* Fix elastic beanstalk solution stack name [PR #271](https://github.com/aws/aws-xray-sdk-go/pull/271)

Release v1.1.0 (2020-06-08)
================================
### SDK Breaking Changes
* Added Disabling XRay SDK Support. [PR #219](https://github.com/aws/aws-xray-sdk-go/pull/219)

### SDK Enhancements
* Added IMDSv2 Support. [PR #235](https://github.com/aws/aws-xray-sdk-go/pull/235)
* Sanitize query string from url in http client segment [PR #228](https://github.com/aws/aws-xray-sdk-go/pull/228)

Release v1.0.1 (2020-04-28)
================================
### SDK Enhancements
* Random value generator only used by SDK. [PR #183](https://github.com/aws/aws-xray-sdk-go/pull/183)

### SDK Bugs
* Fixed deadlock issue for non reported segments. [PR #223](https://github.com/aws/aws-xray-sdk-go/pull/223)

Release v1.0.0 (2020-04-16)
================================
### SDK Breaking Changes
* Removed plugins under "github.com/aws/aws-xray-sdk-go/plugins/“ directory and removed deprecated xray.SQL API (sql.go file). [PR #215](https://github.com/aws/aws-xray-sdk-go/pull/215)
* Added Dummy flag support to reduce operation of non sampled traces. [PR #194](https://github.com/aws/aws-xray-sdk-go/pull/194)

### SDK Enhancements
* Benchmark improvements to remove error logs. [PR #210](https://github.com/aws/aws-xray-sdk-go/pull/210)
* Updates golangci-lint version. [PR #213](https://github.com/aws/aws-xray-sdk-go/pull/213)
* Benchmark instructions to run benchamrk suite, cpu profiling and memory profiling of SDK. [PR #214](https://github.com/aws/aws-xray-sdk-go/pull/214)

### SDK Bugs
* Fixes bug in reservoir test cases. [PR #212](https://github.com/aws/aws-xray-sdk-go/pull/212)

Release v1.0.0-rc.15 (2020-03-11)
================================
### SDK Breaking Changes
* Custom SQL driver. [PR #169](https://github.com/aws/aws-xray-sdk-go/pull/169)

### SDK Enhancements
* Efficient implementation of wildcard matching. [PR #149](https://github.com/aws/aws-xray-sdk-go/pull/149)
* Whitelisting Sagemaker runtime InvokeEndpoint operation. [PR #154](https://github.com/aws/aws-xray-sdk-go/pull/154/files)
* Added context missing environment variable support. [PR #161](https://github.com/aws/aws-xray-sdk-go/pull/161)
* Added stale bot support for Github Repository. [PR #162](https://github.com/aws/aws-xray-sdk-go/pull/162)
* Upgrade golangci-lint version. [PR #166](https://github.com/aws/aws-xray-sdk-go/pull/166)
* Fixes golint warnings. [PR #170](https://github.com/aws/aws-xray-sdk-go/pull/170)
* Added support for Git Actions. [PR #172](https://github.com/aws/aws-xray-sdk-go/pull/172)
* README update for presign request. [PR #176](https://github.com/aws/aws-xray-sdk-go/pull/176)
* Fix data races in testing. [PR #177](https://github.com/aws/aws-xray-sdk-go/pull/177)
* Fixes sampling issue in calling BeginSegment API directly. [PR #187](https://github.com/aws/aws-xray-sdk-go/pull/187)
* Captures error type from panic. [PR #195](https://github.com/aws/aws-xray-sdk-go/pull/195)
* Upgrades Sqlmock to 1.4.1 . [PR #190](https://github.com/aws/aws-xray-sdk-go/pull/190)
* Fixes data race in default sampling strategy. [PR #196](https://github.com/aws/aws-xray-sdk-go/pull/196)
* Added benchmark support to data races and performance issues. [PR #197](https://github.com/aws/aws-xray-sdk-go/pull/197)
* Upgrades Travis CI to add Go 1.14 . [PR #198](https://github.com/aws/aws-xray-sdk-go/pull/198)
* Fixes data race in emitter. [PR #200](https://github.com/aws/aws-xray-sdk-go/pull/200)

### SDK Bugs
* Fixes break logging tools. [PR #185](https://github.com/aws/aws-xray-sdk-go/pull/185)
* Fixes memory leak in BeginSegment method. [PR #156](https://github.com/aws/aws-xray-sdk-go/pull/156)

Release v1.0.0-rc.14 (2019-09-03)
================================
### SDK Enhancements
* Fixing bi-directional locking for parent-child segments and modifying lock on Segment struct to RWMutex [PR #140](https://github.com/aws/aws-xray-sdk-go/pull/140)

Release v1.0.0-rc.13 (2019-07-18)
================================
### SDK New Features
* Support capturing AWS SNS Topic ARN. [PR #132](https://github.com/aws/aws-xray-sdk-go/pull/132)
* Add `xray.AWSSession` to install handlers on session. [PR #97](https://github.com/aws/aws-xray-sdk-go/pull/97)

### SDK Breaking Changes
* Remove Glide(dependency management tool) support. [PR #129](https://github.com/aws/aws-xray-sdk-go/pull/129)

### SDK Enhancements
* Update README regarding Lambda use cases. [PR #128](https://github.com/aws/aws-xray-sdk-go/pull/128)
* Fix a bug to close in_progress response subsegment. [PR #125](https://github.com/aws/aws-xray-sdk-go/pull/125)
* Move mutex to private member for sampling. [PR #123](https://github.com/aws/aws-xray-sdk-go/pull/123)
* Fix format after running Lints. [PR #117](https://github.com/aws/aws-xray-sdk-go/pull/117)

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
