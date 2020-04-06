# Benchmark Instructions for AWS X-Ray Go SDK
AWS X-Ray Go SDK introduced benchmarks to identify performance bottlenecks of AWS X-Ray Go SDK codebase. Moreover, benchmarks can be used to identify data races and locking issues. Below are the instructions on how to run AWS X-Ray Go SDK benchmarks using Go commands and makefile.

## Run all the benchmarks using Go Command
```
go test -benchmem -run=^$$ -bench=. ./...
```

## Run all the benchmark using makefile
Running below command will generate benchmark_sdk.md for analysis. To avoid excessive logging change the loglevel to LogLevelError.
```
make benchmark_sdk
```
## Run memory profiling of xray package using makefile
Running below command will generate benchmark_xray_mem.md for analysis.
```
make benchmark_xray_mem
```
## Run cpu profiling of xray package using makefile
Running below command will generate benchmark_xray_cpu.md for analysis.
```
make benchmark_xray_cpu
```

