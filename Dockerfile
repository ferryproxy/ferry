FROM docker.io/library/golang:alpine AS builder
WORKDIR /go/src/github.com/ferry-tunnel/ferry/
COPY . .
ENV CGO_ENABLED=0
RUN apk add git && go install ./cmd/controller

FROM docker.io/library/alpine
COPY --from=builder /go/bin/controller /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/controller" ]
