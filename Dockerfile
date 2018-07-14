FROM golang:alpine AS go-build-env

WORKDIR /go/src/github.com/lon9/docker-notify
RUN apk add --no-cache git
ADD . /go/src/github.com/lon9/docker-notify
RUN go get
RUN go build -o /usr/bin/docker-notify

FROM alpine
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=go-build-env /usr/bin/docker-notify .
