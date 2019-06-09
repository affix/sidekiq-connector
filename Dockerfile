FROM golang:1.9.2-alpine
RUN mkdir -p /go/src/github.com/affix/sidekiq-connector
RUN apk -U add curl git && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh && \
    apk del curl
WORKDIR /go/src/github.com/affix/sidekiq-connector

COPY types      types
COPY Gopkg.lock Gopkg.lock
COPY Gopkg.toml Gopkg.toml
COPY main.go    .
RUN dep ensure

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...

# Stripping via -ldflags "-s -w"
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o ./connector

CMD ["./connector"]
