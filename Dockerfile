FROM golang:1 as builder
LABEL maintainer="Joel Messerli <hi.github@peg.nu>"
WORKDIR /go/src/github.com/jmesserli/nx
COPY . .
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/nx .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /go/bin/nx .
COPY --from=builder /go/src/github.com/jmesserli/nx/templates ./templates
CMD ["./nx"]