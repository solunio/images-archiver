# Dockerfile - uses pre-built binary
# Build the binary first with: make build-release

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy pre-built binary (expects 'images-archiver' in build context)
COPY images-archiver* /app/images-archiver

# Ensure binary is executable
RUN chmod +x /app/images-archiver

ENTRYPOINT ["/app/images-archiver"]
