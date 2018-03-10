GO_PKGS = $(shell go list ./... | \grep -v vendor)
GO_SRCS = $(shell find . -type f -name '*.go')

# test runs the lint/fmt and the tests.
.PHONY: test
test: fmt imports
test: ${GO_SRCS}
	go test -cover -race ${GO_PKGS}

.PHONY: fmt
fmt: ${GO_SRCS}
	@echo "Checking go format."
	@echo "gofmt -s -l . | \grep -v vendor"
	@[ -z "$(shell gofmt -s -l . | \grep -v vendor | \tee /dev/stderr)" ]

.PHONY: imports
imports: ${GO_SRCS}
	@echo "Checking go imorts."
	@echo "goimports -l . | \grep -v vendor"
	@[ -z "$(shell goimports -l . | \grep -v vendor | \tee /dev/stderr)" ]
