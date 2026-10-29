FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/app

FROM alpine:3.20
WORKDIR /srv

COPY --from=build /app ./app
COPY web ./web

EXPOSE 8080
CMD ["./app"]
