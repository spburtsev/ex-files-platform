FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/ogen-go/ogen/cmd/ogen@v1.20.3
COPY . .
COPY --from=spec openapi.yaml /spec/openapi.yaml
RUN ogen --target oapi --package oapi --clean /spec/openapi.yaml
RUN CGO_ENABLED=0 go build -o /app .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
COPY --from=build /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
