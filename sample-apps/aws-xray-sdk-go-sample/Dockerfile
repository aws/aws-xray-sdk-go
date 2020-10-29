:# SPDX-License-Identifier: MIT-0

FROM golang:1.14

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

ENV GOPROXY direct

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

EXPOSE 8080

EXPOSE 8000

CMD ["./main"]
