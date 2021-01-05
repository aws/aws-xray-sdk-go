![Test](https://github.com/aws/aws-xray-sdk-go/workflows/Test/badge.svg)

# AWS X-Ray SDK for Go

![Screenshot of the AWS X-Ray console](/images/example.png?raw=true)

## Installing into GOPATH

The AWS X-Ray SDK for Go is compatible with Go 1.9 and above.

Install the SDK using the following command (The SDK's non-testing dependencies will be installed):
Use `go get` to retrieve the SDK to add it to your `GOPATH` workspace:

```
go get github.com/aws/aws-xray-sdk-go
```

To update the SDK, use `go get -u` to retrieve the latest version of the SDK.

```
go get -u github.com/aws/aws-xray-sdk-go
```

If you also want to install SDK's testing dependencies. They can be installed using:

```
go get -u -t github.com/aws/aws-xray-sdk-go/...
```

## Installing using Go Modules

The latest version of the SDK is the recommended version.

If you are using Go 1.11 and above, you can install the SDK using Go Modules (in project's go.mod), like so: 

```
go get github.com/aws/aws-xray-sdk-go
```

To get a different specific release version of the SDK use `@<tag>` in your `go get` command. Also, to get the rc version use this command with the specific version.

```
go get github.com/aws/aws-xray-sdk-go@v1.0.0
```

## Installing using Dep
If you are using Go 1.9 and above, you can also use [Dep](https://github.com/golang/dep) to add the SDK to your application's dependencies.
Using Dep will help your application stay pinned to a specific version of the SDK.

To add the SDK to your application using Dep, run:

```
dep ensure -add github.com/aws/aws-xray-sdk-go
```

## Getting Help

Please use these community resources for getting help. We use the GitHub issues for tracking bugs and feature requests.

* Ask a question in the [AWS X-Ray Forum](https://forums.aws.amazon.com/forum.jspa?forumID=241&start=0).
* Open a support ticket with [AWS Support](http://docs.aws.amazon.com/awssupport/latest/user/getting-started.html).
* If you think you may have found a bug, please open an [issue](https://github.com/aws/aws-xray-sdk-go/issues/new).

## Opening Issues

If you encounter a bug with the AWS X-Ray SDK for Go we would like to hear about it. Search the [existing issues](https://github.com/aws/aws-xray-sdk-go/issues) and see if others are also experiencing the issue before opening a new issue. Please include the version of AWS X-Ray SDK for Go, AWS SDK for Go, Go language, and OS you’re using. Please also include repro case when appropriate.

The GitHub issues are intended for bug reports and feature requests. For help and questions regarding the use of the AWS X-Ray SDK for Go please make use of the resources listed in the [Getting Help](https://github.com/aws/aws-xray-sdk-go#getting-help) section. Keeping the list of open issues lean will help us respond in a timely manner.

## Documentation

The [developer guide](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-go.html) provides in-depth guidance on using the AWS X-Ray service and the AWS X-Ray SDK for Go.

See [aws-xray-sdk-go-sample](https://github.com/aws-samples/aws-xray-sdk-go-sample) for a sample application that provides example of tracing SQL queries, incoming and outgoing request. Follow [README instructions](https://github.com/aws-samples/aws-xray-sdk-go-sample/blob/master/README.md) in that repository to get started with sample application.

## Quick Start

**Configuration**

```go
import "github.com/aws/aws-xray-sdk-go/xray"

func init() {
  xray.Configure(xray.Config{
    DaemonAddr:       "127.0.0.1:2000", // default
    ServiceVersion:   "1.2.3",
  })
}
```
***Logger***

xray uses an interface for its logger:

```go
type Logger interface {
  Log(level LogLevel, msg fmt.Stringer)
}

const (
  LogLevelDebug LogLevel = iota + 1
  LogLevelInfo
  LogLevelWarn
  LogLevelError
)
```

The default logger logs to [stdout](https://golang.org/pkg/syscall/#Stdout) at "info" and above. To change the logger, call `xray.SetLogger(myLogger)`. There is a default logger implementation that writes to an `io.Writer` from a specified minimum log level. For example, to log to stderr at "error" and above:

```go
xray.SetLogger(xraylog.NewDefaultLogger(os.Stderr, xraylog.LogLevelError))
```

Note that the `xray.Config{}` fields `LogLevel` and `LogFormat` are deprecated starting from version `1.0.0-rc.10` and no longer have any effect.

***Plugins***

Plugins can be loaded conditionally at runtime. For this purpose, plugins under "github.com/aws/aws-xray-sdk-go/awsplugins/" have an explicit `Init()` function. Customer must call this method to load the plugin:

```go
import (
  "os"

  "github.com/aws/aws-xray-sdk-go/awsplugins/ec2"
  "github.com/aws/aws-xray-sdk-go/xray"
)

func init() {
  // conditionally load plugin
  if os.Getenv("ENVIRONMENT") == "production" {
    ec2.Init()
  }

  xray.Configure(xray.Config{
    ServiceVersion:   "1.2.3",
  })
}
```

**Start a custom segment/subsegment**
Note that customers using xray.BeginSegment API directly will only be able to evaluate sampling rules based on service name.

```go
  // Start a segment
  ctx, seg := xray.BeginSegment(context.Background(), "service-name")
  // Start a subsegment
  subCtx, subSeg := xray.BeginSubsegment(ctx, "subsegment-name")
  // ...
  // Add metadata or annotation here if necessary
  // ...
  subSeg.Close(nil)
  // Close the segment
  seg.Close(nil)
```

**Disabling XRay Tracing**

XRay tracing can be disabled by setting up environment variable `AWS_XRAY_SDK_DISABLED` . Disabling XRay can be useful for specific use case like if customer wants to stop tracing in their test environment they can do so just by setting up the environment variable.



```go
  // Set environment variable TRUE to disable XRay
  os.Setenv("AWS_XRAY_SDK_DISABLED", "TRUE")
```

**Capture**

```go
func criticalSection(ctx context.Context) {
  // This example traces a critical code path using a custom subsegment
  xray.Capture(ctx, "MyService.criticalSection", func(ctx1 context.Context) error {
    var err error

    section.Lock()
    result := someLockedResource.Go()
    section.Unlock()

    xray.AddMetadata(ctx1, "ResourceResult", result)
  })
}
```

**HTTP Handler**

```go
func main() {
  http.Handle("/", xray.Handler(xray.NewFixedSegmentNamer("myApp"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello!"))
  })))
  http.ListenAndServe(":8000", nil)
}
```

**HTTP Client**

```go
func getExample(ctx context.Context) ([]byte, error) {
    resp, err := ctxhttp.Get(ctx, xray.Client(nil), "https://aws.amazon.com/")
    if err != nil {
      return nil, err
    }
    return ioutil.ReadAll(resp.Body)
}
```

**AWS**

```go
sess := session.Must(session.NewSession())
dynamo := dynamodb.New(sess)
xray.AWS(dynamo.Client)
dynamo.ListTablesWithContext(ctx, &dynamodb.ListTablesInput{})
```

**S3**

`aws-xray-sdk-go` does not currently support [`*Request.Presign()`](https://docs.aws.amazon.com/sdk-for-go/api/aws/request/#Request.Presign) operations and will panic if one is encountered.  This results in an error similar to: 

`panic: failed to begin subsegment named 's3': segment cannot be found.`

If you encounter this, you can set `AWS_XRAY_CONTEXT_MISSING` environment variable to `LOG_ERROR`.  This will instruct the SDK to log the error and continue processing your requests.

```go
os.Setenv("AWS_XRAY_CONTEXT_MISSING", "LOG_ERROR")
```

**SQL**

Any `database/sql` calls can be traced with X-Ray by replacing the `sql.Open` call with `xray.SQLContext`. It is recommended to use URLs instead of configuration strings if possible.

```go
func main() {
  db, err := xray.SQLContext("postgres", "postgres://user:password@host:port/db")
  row, err := db.QueryRowContext(ctx, "SELECT 1") // Use as normal
}
```

**Lambda**

```
For Lambda support use version v1.0.0-rc.1 and higher
```

If you are using the AWS X-Ray Go SDK inside a Lambda function, there will be a FacadeSegment inside the Lambda context.  This allows you to instrument your Lambda function using `Configure`, `Capture`, `HTTP Client`, `AWS`, `SQL` and `Custom Subsegments` usage.  `Segment` operations are not supported.

```go
func HandleRequest(ctx context.Context, name string) (string, error) {
    sess := session.Must(session.NewSession())
    dynamo := dynamodb.New(sess)
    xray.AWS(dynamo.Client)
    input := &dynamodb.PutItemInput{
        Item: map[string]*dynamodb.AttributeValue{
            "12": {
                S: aws.String("example"),
            },
        },
        TableName: aws.String("xray"),
    }
    _, err := dynamo.PutItemWithContext(ctx, input)
    if err != nil {
        return name, err
    }
    
    _, err = ctxhttp.Get(ctx, xray.Client(nil), "https://www.twitch.tv/")
    if err != nil {
        return name, err
    }
    
    _, subseg := xray.BeginSubsegment(ctx, "subsegment-name")
    subseg.Close(nil)
    
    db := xray.SQLContext("postgres", "postgres://user:password@host:port/db")
    row, _ := db.QueryRow(ctx, "SELECT 1")
    
    return fmt.Sprintf("Hello %s!", name), nil
}
```

## License

The AWS X-Ray SDK for Go is licensed under the Apache 2.0 License. See LICENSE and NOTICE.txt for more information.
