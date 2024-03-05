FROM golang:1.22-alpine3.19 as builder
COPY go.mod go.mod
COPY go.sum go.sum
COPY cmd/links cmd/links
COPY services/links services/links
COPY pkg/logging pkg/logging
RUN go build -v -o /links cmd/links/main.go

FROM alpine:3.19
COPY --from=builder /links .
ENTRYPOINT [ "./links" ]