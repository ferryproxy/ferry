FROM docker.io/library/golang:alpine AS builder
WORKDIR /go/src/github.com/ferry-tunnel/ferry/
COPY . .
ENV CGO_ENABLED=0
RUN apk add git
RUN go install ./cmd/ferry-tunnel

FROM ghcr.io/wzshiming/bridge/bridge:v0.8.5
COPY --from=builder /go/bin/ferry-tunnel /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/ferry-tunnel" ]
