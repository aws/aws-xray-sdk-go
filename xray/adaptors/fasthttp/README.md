# Instrumentation for fasthttp
## Usage:


First import xrayfastttp:
``go
import xrayfasthttp "github.com/aws/aws-xray-sdk-go/xray/adaptors/fasthttp"
``

Example with Router:
``go
        r := router.New()
        r.GET("/", xrayfasthttp.Handler(xray.NewFixedSegmentNamer("index"), Index))
        r.GET("/hello/{name}", xrayfasthttp.Handler(xray.NewFixedSegmentNamer("hello"), Hello))

        log.Fatal(fasthttp.ListenAndServe(":8080", r.Handler))

``
`
