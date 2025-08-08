# syntax=docker/dockerfile:1
FROM golang:1.22 as build
WORKDIR /src
COPY go.mod ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/app /app
ENV HTTP_ADDR=:8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app"]

