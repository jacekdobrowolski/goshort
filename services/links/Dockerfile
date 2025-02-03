FROM golang:1.23.5-alpine3.20 as builder
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY cmd/links cmd/links
COPY pkg pkg
COPY services/links services/links
RUN --mount=type=cache,target="/root/.cache/go-build" go build -v -ldflags="-w -s" -o /links cmd/links/main.go
RUN --mount=type=cache,target="/root/.cache/go-build" go test ./services/links -c -o /links.test

FROM alpine:3.20 as test
COPY --from=builder /links.test .
RUN ./links.test -test.v

FROM scratch as final
COPY --from=builder /links .
ENTRYPOINT [ "./links" ]
