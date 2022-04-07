# Compile stage
FROM golang:1.18 AS build-env

# Build Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

ADD . /builder
WORKDIR /builder/cmd/sa-collector

RUN go build -gcflags="all=-N -l" -o /sa-collector

# Final stage
FROM satan-dbg-base:latest

EXPOSE 8000 40000

WORKDIR /
COPY --from=build-env /go/bin/dlv /
COPY --from=build-env /sa-collector /

CMD ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "exec", "/sa-collector"]
