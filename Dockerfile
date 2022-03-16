FROM golang:1.17-buster AS build

WORKDIR /app

COPY . ./
RUN go mod download

RUN go build -o system-health-bot ./cmd/main.go

##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build ["/app/system-health-bot", "/app/.env", "/"]

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/system-health-bot"]