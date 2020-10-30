# SampleApp for AWS X-Ray Go SDK

This repository contains sample app to show the tracing use case of aws-xray-sdk-go. The SampleApp contains example of tracing aws sdk calls like list all s3 buckets. Moreover, it contains tracing upstream HTTP request. 

## Prerequirements

* Should have XRay daemon or AOC with xray receiver installed, and running in order to see traces on the AWS XRay console

## Requst route path

This application contains 2 paths
```
/aws-sdk-call
/outgoing-http-call
```

## Opening Issues

If you encounter a bug specifically with the SampleApp for AWS X-Ray Go SDK should be reported to this repository whereas bugs with the X-Ray Go SDK should be reported [here](https://github.com/aws/aws-xray-sdk-go/issues). 

## License

This library is licensed under the MIT-0 License. See the LICENSE file.
