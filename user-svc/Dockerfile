FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml && CGO_ENABLED=0 go build -o /out/user-service ./cmd/api

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/user-service /user-service
EXPOSE 8080
ENTRYPOINT ["/user-service"]
