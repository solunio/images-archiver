# Images Archiver

[![CI](https://github.com/solunio/images-archiver/actions/workflows/ci.yml/badge.svg)](https://github.com/solunio/images-archiver/actions/workflows/ci.yml)

A tool to create docker-archive files containing one or more container images with efficient layer deduplication.

## Features

- ✨ Download and archive multiple container images
- 🚀 Efficient layer deduplication through local OCI cache
- 📦 Compatible with both `docker load` and `podman load`
- 📤 Output to file or stdout (supports piping to compression tools)
- 🔐 Supports authentication via `~/.docker/config.json`
- 🔇 Optional quiet mode to disable progress output

## Installation

### From Container Image

Pull the pre-built image from GitHub Container Registry:

```bash
docker pull ghcr.io/solunio/images-archiver:latest

# Or use a specific version
docker pull ghcr.io/solunio/images-archiver:v1.0.0
```

### From Source

```bash
go install github.com/solunio/images-archiver@latest
```

### From Release

Download pre-built binaries from the [releases page](https://github.com/solunio/images-archiver/releases).

## Usage

```bash
# Basic usage - output to stdout
images-archiver nginx:latest alpine:latest > archive.tar

# Pipe to gzip for compression
images-archiver nginx:latest | gzip > archive.tar.gz

# Output to a file
images-archiver -o myimages.tar redis:7 postgres:15

# Quiet mode (disable progress output) and pipe to pigz (parallel gzip)
images-archiver -q registry.example.com/app:v1.0.0 | pigz > archive.tar.gz

# Overwrite existing file with force flag
images-archiver -f -o existing.tar nginx:latest
```

### Options

- `-o <filename>`: Output archive file name (default: stdout)
- `-c <path>`: Cache directory for layer deduplication (default: temp directory)
- `-f`: Overwrite output file if it already exists
- `-q`: Disable layer download progress output (quiet mode)

## Examples

```bash
# Archive multiple images
images-archiver nginx:latest alpine:latest redis:7

# Use custom cache directory
images-archiver -c /var/cache/images nginx:latest

# Archive private registry images (uses credentials from ~/.docker/config.json)
images-archiver registry.example.com/private/app:latest

# Create compressed archive with quiet mode
images-archiver -q \
  nginx:latest \
  postgres:15 \
  redis:7 \
  alpine:latest | gzip -9 > images.tar.gz

# Overwrite existing archive file
images-archiver -f -o myimages.tar nginx:latest redis:7
```

## Loading Images

The created archive is compatible with both Docker and Podman:

```bash
# Docker
docker load -i archive.tar

# Podman
podman load -i archive.tar

# From compressed archive
gunzip -c archive.tar.gz | docker load
```

## How It Works

1. **First Stage**: Downloads all specified images to a local OCI cache directory
2. **Second Stage**: Copies images from the cache to a docker-archive format
3. **Deduplication**: Shared layers between images are only downloaded once and stored once in the archive
4. **Progress**: Shows download progress for each layer (can be disabled with `-q` flag)

## Building

```bash
# Build using Makefile (recommended)
make build

# Build optimized release binary (smaller size, no debug symbols)
make build-release

# Run tests
make test

# Run tests with coverage and race detection (for CI)
make test-ci

# Clean build artifacts
make clean

# Or build directly with Go
go build -tags "exclude_graphdriver_btrfs,containers_image_openpgp" -o images-archiver .

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -tags "exclude_graphdriver_btrfs,containers_image_openpgp" -o images-archiver-linux-amd64 .
```

### Docker Images

```bash
# Build Docker image (compiles binary then builds image)
make build-docker

# Or build manually
make build-release
docker build -t images-archiver .

# Run locally built image
docker run --rm images-archiver nginx:latest alpine:latest > archive.tar
```

**Note**: The build tags are used to avoid requiring system dependencies:
- `exclude_graphdriver_btrfs`: Avoids BTRFS development headers (`btrfs-progs-devel`)
- `containers_image_openpgp`: Uses pure Go crypto instead of GPG/GPGME (`gpgme-devel`)

## Releasing

**Important**: Releases are created on separate `release/` branches, **NEVER on the main branch**.

### To create a release:

1. Create a release branch from main:
   ```bash
   git checkout -b release/v1.0.0
   ```

2. Make any necessary release preparations (update version numbers, changelog, etc.)

3. Commit your changes and push the release branch:
   ```bash
   git push origin release/v1.0.0
   ```

4. Create and push the release tag:
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

This will trigger the CI/CD pipeline to build and publish the release artifacts.

## Requirements

- Go 1.25 or later

## License

This project is primarily licensed under the [Blue Oak Model License 1.0.0](https://blueoakcouncil.org/license/1.0.0) - see the [LICENSE](LICENSE) file for details.

### Third-Party Code

This project includes code from third-party sources:

- `third_party/podman/utils_linux.go` contains code copied from [Podman](https://github.com/containers/podman), which is licensed under the [Apache License 2.0](LICENSE-APACHE-2.0). The original copyright and license notice are preserved in that file.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

