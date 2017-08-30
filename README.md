# AWS X-Ray SDK for Go <sup><sup><sup>(beta)</sup></sup></sup>

![Screenshot of the AWS X-Ray console](/images/example_trace.png?raw=true)

## Installing

The AWS X-Ray SDK for Go is compatible with Go 1.7 and above.

Install the SDK using the following command:

```
go get -u github.com/aws/aws-xray-sdk-go
```

Some external dependencies are required. They may be installed using:

```
glide install
```

Or by individually calling `go get` for each of the required dependencies.

## Getting Help

Please use these community resources for getting help. We use the GitHub issues for tracking bugs and feature requests.

* Ask a question on [StackOverflow](http://stackoverflow.com/) and tag it with the [`aws-xray`](http://stackoverflow.com/questions/tagged/aws-xray) tag.
* Open a support ticket with [AWS Support](http://docs.aws.amazon.com/awssupport/latest/user/getting-started.html).
* If you think you may have found a bug, please open an [issue](https://github.com/aws/aws-xray-sdk-go/issues/new).

## Opening Issues

If you encounter a bug with the AWS X-Ray SDK for Go we would like to hear about it. Search the [existing issues](https://github.com/aws/aws-xray-sdk-go/issues) and see if others are also experiencing the issue before opening a new issue. Please include the version of AWS X-Ray SDK for Go, AWS SDK for Go, Go language, and OS you’re using. Please also include repro case when appropriate.

The GitHub issues are intended for bug reports and feature requests. For help and questions with using AWS SDK for Go please make use of the resources listed in the [Getting Help](https://github.com/aws/aws-xray-sdk-go#getting-help) section. Keeping the list of open issues lean will help us respond in a timely manner.

## Documentation

The [developer guide](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-go.html) proivdes in-depth guidance on using the AWS X-Ray service and the AWS X-Ray SDK for Go.

#### Quick Start

**Configuration**

```go
import (
  "context"

  "github.com/aws/aws-xray-sdk-go/xray"

  // Importing the plugins enables collection of AWS resource information at runtime
  _ "github.com/aws/aws-xray-sdk-go/plugins/ec2"
  _ "github.com/aws/aws-xray-sdk-go/plugins/beanstalk"
  _ "github.com/aws/aws-xray-sdk-go/plugins/ecs"
)

func init() {
  xray.Configure(xray.Config{
    DaemonAddr:       "127.0.0.1:2000", // default
    LogLevel:         "info",           // default
    ServiceVersion:   "1.2.3",
  })
}
```

**Capture**

```go
func criticalSection(ctx context.Context) {
  // this example traces a critical code path using a custom subsegment
  xray.Capture(ctx, "MyService.criticalSection", func(ctx1 context.Context) error {
    var err error

    section.Lock()
    result := someLockedResource.Go()
    section.Unlock()

    xray.AddMetadata(ctx1, "ResourceResult", result)
  })
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

**SQL**

Any `db/sql` calls can be traced with XRay by replacing the `sql.Open` call with `xray.SQL`. It is recommended to use URLs instead of configuration strings if possible.

```go
func main() {
  db := xray.SQL("postgres", "postgres://user:password@host:port/db")
  row, _ := db.QueryRow("SELECT 1") // Use as normal
}
```

## License

The AWS X-Ray SDK for Go is licensed under the Apache 2.0 License. See LICENSE and NOTICE.txt for more information.
