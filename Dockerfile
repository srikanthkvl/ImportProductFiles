FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/rest ./cmd/rest && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/cli ./cmd/cli && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/grpc ./cmd/grpc

FROM gcr.io/distroless/base-debian12
COPY --from=build /bin/rest /bin/rest
COPY --from=build /bin/cli /bin/cli
COPY --from=build /bin/grpc /bin/grpc
# Config and mapping are mounted via compose; provide defaults for local docker build
COPY customer_map.json /customer_map.json
COPY config.json /app/config.json
COPY api/swagger.json /app/api/swagger.json
ENV CONFIG_PATH=/app/config.json
ENTRYPOINT ["/bin/rest"]
EXPOSE 8080 9090


