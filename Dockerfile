FROM golang:1.9 as builder
WORKDIR /go/src/github.com/proofpoint/prom-to-kairosdb

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/prom-to-kairosdb

FROM alpine:3.6
RUN mkdir /opt
RUN apk add --no-cache ca-certificates
WORKDIR /opt/
COPY --from=builder /go/src/github.com/proofpoint/prom-to-kairosdb/bin/prom-to-kairosdb .
ENTRYPOINT ["./prom-to-kairosdb"]
