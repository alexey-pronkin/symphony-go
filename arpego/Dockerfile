FROM golang:1.24.1-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/arpego ./cmd/arpego

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/arpego /usr/local/bin/arpego

EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/arpego"]
