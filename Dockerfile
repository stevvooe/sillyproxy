FROM golang:1.7-alpine

COPY . /go/src/github.com/stevvooe/sillyproxy
RUN go install github.com/stevvooe/sillyproxy

ENTRYPOINT ["/go/bin/sillyproxy"]`
