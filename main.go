package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.podman.io/image/v5/types"
)

func main() {
	// Define flags
	outputFileFlag := flag.String("o", "", "Output archive file name (default: stdout)")
	cacheDirFlag := flag.String("c", "", "Cache directory for layer deduplication (default: temp directory)")
	forceFlag := flag.Bool("f", false, "Overwrite output file if it already exists")
	noProgressFlag := flag.Bool("q", false, "Disable layer download progress output (quiet mode)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] IMAGE [IMAGE...]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Create a docker-archive containing one or more container images.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s nginx:latest alpine:latest > archive.tar\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s nginx:latest | gzip > archive.tar.gz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -o myimages.tar redis:7 postgres:15\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -f -o existing.tar app:latest\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -q registry.example.com/app:v1.0.0 | pigz > archive.tar.gz\n\n", os.Args[0])
	}
	flag.Parse()

	// Get image references from command-line arguments
	imageRefs := flag.Args()

	// If no arguments provided, show usage and exit
	if len(imageRefs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No images specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Determine cache directory
	cacheDir := *cacheDirFlag
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), fmt.Sprintf("image-creator-cache-%d", os.Getpid()))
	}

	// Call run with dereferenced flags
	if err := run(imageRefs, *outputFileFlag, cacheDir, *noProgressFlag, *forceFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Only print success message if writing to a file (not stdout)
	if *outputFileFlag != "" {
		fmt.Fprintf(os.Stderr, "Successfully created docker-archive: %s (compatible with both 'docker load -i' and 'podman load -i')\n", *outputFileFlag)
	}
}

func run(imageRefs []string, outputFile string, cacheDir string, noProgress bool, force bool) error {
	ctx := context.Background()
	// Use default system context (handles TLS, credentials from ~/.docker/config.json, etc.)
	sysCtx := &types.SystemContext{}

	// --- 1. Setup OCI Cache Directory ---
	if err := setupCacheDir(cacheDir); err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(cacheDir); err != nil {
			logrus.Errorf("Failed to clean up cache directory: %v", err)
		}
	}()

	// --- 2. Setup signature policy ---
	policyContext, err := createPolicyContext()
	if err != nil {
		return err
	}

	// --- 3. Download images to shared cache (sequential with blob deduplication) ---
	if err := downloadImagesToCache(ctx, imageRefs, cacheDir, noProgress, sysCtx, policyContext); err != nil {
		return err
	}

	// --- 4. Second Stage: Copy from cache to docker archive ---
	return createArchiveFromCache(ctx, imageRefs, outputFile, cacheDir, force, sysCtx, policyContext)
}
