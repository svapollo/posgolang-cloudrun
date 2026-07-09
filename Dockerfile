FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM gcr.io/distroless/static-debian12

WORKDIR /

COPY --from=build /app/app /app

ENV PORT=8080
EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app"]
