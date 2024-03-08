FROM golang:1.22.1-alpine3.19 as builder
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY cmd/links cmd/links
COPY pkg pkg
COPY services/links services/links
RUN --mount=type=cache,target="/root/.cache/go-build" go build -v -o /links cmd/links/main.go

FROM alpine:3.19
COPY --from=builder /links .
ENTRYPOINT [ "./links" ]