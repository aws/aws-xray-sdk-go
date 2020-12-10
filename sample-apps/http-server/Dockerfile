FROM golang:1.14

WORKDIR /app
COPY . .
ENV GOPROXY direct

RUN go mod download
RUN cd sample-apps/http-server/ && go install .

# Build Golang sample app
RUN cd sample-apps/http-server/ && go build -o application .

EXPOSE 8080

# Entrypoint
CMD ["sample-apps/http-server/application"]
