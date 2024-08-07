FROM golang:1.22.6-alpine3.20 as builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build -o /pingo

FROM scratch
COPY --from=builder /pingo /
ENTRYPOINT [ "/pingo" ]
