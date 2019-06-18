FROM golang:1.12 as builder
LABEL maintainer="Joel Messerli <hi.github@peg.nu>"
WORKDIR /go/src/github.com/jmesserli/netbox-to-bind
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/netbox-to-bind .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/bin/netbox-to-bind .
CMD ["./netbox-to-bind"] 