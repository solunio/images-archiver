# Dockerfile - uses pre-built binary
# Build the binary first with: make build-release

FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Copy pre-built binary (expects 'images-archiver' in build context)
COPY images-archiver* /usr/bin/images-archiver

# Ensure binary is executable
RUN chmod +x /usr/bin/images-archiver

ENTRYPOINT ["/usr/bin/images-archiver"]
