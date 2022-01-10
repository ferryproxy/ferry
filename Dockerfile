FROM golang:alpine AS builder
WORKDIR /go/src/github.com/ferry-tunnel/ferry/
COPY . .
ENV CGO_ENABLED=0
RUN go install ./cmd/controller

FROM alpine
COPY --from=builder /go/bin/controller /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/controller" ]