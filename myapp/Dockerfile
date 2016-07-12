FROM golang:1.7-alpine

COPY . /go/src/github.com/stevvooe/sillyproxy/myapp
RUN go install github.com/stevvooe/sillyproxy/myapp

EXPOSE 8080
ENTRYPOINT ["/go/bin/myapp"]
