GO_VET_CMD      = go tool vet --all -shadow
GO_DIRS         = $(shell ls -d */ | \grep -v vendor)
GO_PKGS         = $(shell go list ./... | \grep -v vendor)
GO_SRCS         = $(shell find . -type f -name '*.go')

# assets removes the existing bindata.go and regenerates it.
.PHONY          : assets
assets          : clean resources/bindata.go

# test runs the lint/vet/fmt and the tests.
.PHONY          : test
test            : vet lint fmt imports
test            : resources/bindata.go ${GO_SRCS}
		go test -cover -race ${GO_PKGS}

.PHONY          : vet
vet             : ${GO_SRCS}
		@echo "Checking go vet."
		${GO_VET_CMD} $(addprefix ./, ${GO_DIRS})

# TODO: Generate properly linted bindata?
.PHONY          : lint
lint            : ${GO_SRCS}
		@echo "Checking go lint."
		golint -set_exit_status $(filter-out resources/,${GO_DIRS})

.PHONY          : fmt
fmt             : ${GO_SRCS}
		@echo "Checking go format."
		@echo "gofmt -s -l . | \grep -v vendor"
		@[ -z "$(shell gofmt -s -l . | \grep -v vendor | \tee /dev/stderr)" ]

.PHONY          : imports
imports         : ${GO_SRCS}
		@echo "Checking go imorts."
		@echo "goimports -l . | \grep -v vendor"
		@[ -z "$(shell goimports -l . | \grep -v vendor | \tee /dev/stderr)" ]

.PHONY          : clean
clean           :
		rm -f resources/bindata.go

# generate & format bindata.go.
resources/bindata.go    : $(shell find resources -type f -name '*.json')
			rm -f $@
			go-bindata -o resources/bindata.go -pkg resources resources/
			goimports -w resources/bindata.go
			gofmt -w -s resources/bindata.go
