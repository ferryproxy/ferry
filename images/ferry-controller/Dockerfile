FROM docker.io/library/golang:alpine AS builder
WORKDIR /go/src/github.com/ferry-tunnel/ferry/
COPY . .
ENV CGO_ENABLED=0
RUN apk add git
RUN go install ./cmd/ferry-controller

FROM docker.io/library/alpine
COPY --from=builder /go/bin/ferry-controller /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/ferry-controller" ]
