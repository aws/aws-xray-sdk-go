SDK_BASE_FOLDERS=$(shell ls -d */ | grep -v vendor)
GO_VET_CMD=go tool vet --all -shadow

assets:
	rm resources/bindata.go
	go-bindata -o resources/bindata.go -pkg resources resources/

vet:
	${GO_VET_CMD} ${SDK_BASE_FOLDERS}

lint:
	golint ${SDK_BASE_FOLDERS}

test::
	go test -cover `go list ./... | grep -v vendor`

test-with-race: test
	go test -cover -race `go list ./... | grep -v vendor`

fmt:
	go fmt `go list ./... | grep -v vendor`

golangci-lint:
	golangci-lint run

# run all the benchmarks of X-Ray SDK (to minimize logging set loglevel to LogLevelError)
benchmark_sdk:
	echo "\`\`\`" > benchmark/benchmark_sdk.md
	go test -v -benchmem -run=^$$ -bench=. ./... >> benchmark/benchmark_sdk.md
	echo >> benchmark/benchmark_sdk.md
	echo "\`\`\`" >> benchmark/benchmark_sdk.md

# Profiling memory (xray package only and to minimize logging set loglevel to LogLevelError)
benchmark_xray_mem:
	go test -benchmem -run=^$$ -bench=. ./xray -memprofile=benchmark/benchmark_xray_mem.profile

	echo "\`\`\`go" > benchmark/benchmark_xray_mem.md
	echo >> benchmark/benchmark_xray_mem.md
	echo "top" | go tool pprof -sample_index=alloc_objects xray.test benchmark/benchmark_xray_mem.profile >> benchmark/benchmark_xray_mem.md
	echo >> benchmark/benchmark_xray_mem.md
	echo "top -cum" | go tool pprof -sample_index=alloc_objects xray.test benchmark/benchmark_xray_mem.profile >> benchmark/benchmark_xray_mem.md
	echo >> benchmark/benchmark_xray_mem.md
	echo "list xray" | go tool pprof -sample_index=alloc_objects xray.test benchmark/benchmark_xray_mem.profile >> benchmark/benchmark_xray_mem.md
	echo >> Benchmark/benchmark_xray_mem.md
	echo "\`\`\`" >> benchmark/benchmark_xray_mem.md

	rm xray.test
	rm benchmark/benchmark_xray_mem.profile

# profiling cpu (xray package only and to minimize logging set loglevel to LogLevelError)
benchmark_xray_cpu:
	go test -benchmem -run=^$$ -bench=. ./xray -cpuprofile=benchmark/benchmark_xray_cpu.profile

	echo "\`\`\`go" > benchmark/benchmark_xray_cpu.md
	echo >> benchmark/benchmark_xray_cpu.md
	echo "top" | go tool pprof xray.test benchmark/benchmark_xray_cpu.profile >> benchmark/benchmark_xray_cpu.md
	echo >> benchmark/benchmark_xray_cpu.md
	echo "top -cum" | go tool pprof xray.test benchmark/benchmark_xray_cpu.profile >> benchmark/benchmark_xray_cpu.md
	echo >> benchmark/benchmark_xray_cpu.md
	echo "list xray" | go tool pprof xray.test benchmark/benchmark_xray_cpu.profile >> benchmark/benchmark_xray_cpu.md
	echo >> benchmark/benchmark_xray_cpu.md
	echo "\`\`\`" >> benchmark/benchmark_xray_cpu.md

	rm xray.test
	rm benchmark/benchmark_xray_cpu.profile