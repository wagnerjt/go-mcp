# https://github.com/GoogleContainerTools/distroless/blob/main/examples/go/Dockerfile
ARG GO_VERSION=1.24.1

FROM golang:${GO_VERSION} as build

WORKDIR /go/src/app
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static-debian12

COPY --from=build /go/bin/app /

CMD ["/app"]