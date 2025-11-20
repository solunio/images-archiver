package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/oci/layout"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"
)

// setupCacheDir creates and prepares the cache directory
func setupCacheDir(cacheDir string) error {
	// Clean up any previous cache
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clean cache directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	return nil
}

// downloadImagesToCache downloads all images to an OCI layout cache directory sequentially.
// The OCI layout format supports multiple images in a single directory with automatic
// blob deduplication - if multiple images share the same layers (e.g., same base image),
// those layers are only downloaded once and stored by their content hash.
//
// NOTE: We initially attempted to implement parallel downloads using a worker pool to
// speed up the process. However, this approach conflicted with blob deduplication because
// copy.Image() is not safe to call concurrently when writing to the same destination
// directory. To avoid concurrency errors, we would have needed to download each image
// to a separate cache directory, which completely defeats the purpose of layer deduplication.
// Sequential downloads are slower but ensure that shared layers are only downloaded once,
// which is critical when archiving many images with common base layers (e.g., 45+ images
// may share the same Ubuntu/Alpine/etc. base, saving significant bandwidth and storage).
func downloadImagesToCache(ctx context.Context, imageRefs []string, cacheDir string, noProgress bool, sysCtx *types.SystemContext, policyContext *signature.PolicyContext) error {
	fmt.Fprintf(os.Stderr, "Stage 1: Downloading %d images to OCI cache (with layer deduplication)...\n", len(imageRefs))

	// Determine where to send progress output
	var progressWriter io.Writer
	if noProgress {
		progressWriter = nil // Disable progress output
	} else {
		progressWriter = os.Stderr // Show progress on stderr
	}

	// Download images sequentially to the OCI layout cache
	// The OCI layout stores blobs in a content-addressable manner (blobs/sha256/...),
	// so shared layers between images are automatically deduplicated.
	var downloadErrors []error
	for i, rawRef := range imageRefs {
		fmt.Fprintf(os.Stderr, "  [%d/%d] Caching: %s\n", i+1, len(imageRefs), rawRef)

		// Parse source reference
		srcRef, err := alltransports.ParseImageName("docker://" + rawRef)
		if err != nil {
			err = fmt.Errorf("invalid source image reference %s: %w", rawRef, err)
			fmt.Fprintf(os.Stderr, "  [%d/%d] ✗ Failed: %s - %v\n", i+1, len(imageRefs), rawRef, err)
			downloadErrors = append(downloadErrors, err)
			continue
		}

		// Create a unique tag for this image in the OCI layout
		// Each image needs a unique tag so they can coexist in the same layout
		cacheTag := fmt.Sprintf("img-%d", i)
		destRef, err := layout.NewReference(cacheDir, cacheTag)
		if err != nil {
			err = fmt.Errorf("failed to create OCI cache reference: %w", err)
			fmt.Fprintf(os.Stderr, "  [%d/%d] ✗ Failed: %s - %v\n", i+1, len(imageRefs), rawRef, err)
			downloadErrors = append(downloadErrors, err)
			continue
		}

		// Copy to OCI cache - blobs are automatically deduplicated by content hash
		_, err = copy.Image(
			ctx,
			policyContext,
			destRef,
			srcRef,
			&copy.Options{
				ReportWriter:   progressWriter, // Show layer download progress (or nil to disable)
				SourceCtx:      sysCtx,
				DestinationCtx: sysCtx,
			},
		)

		if err != nil {
			err = fmt.Errorf("failed to cache image %s: %w", rawRef, err)
			fmt.Fprintf(os.Stderr, "  [%d/%d] ✗ Failed: %s - %v\n", i+1, len(imageRefs), rawRef, err)
			downloadErrors = append(downloadErrors, err)
		} else {
			fmt.Fprintf(os.Stderr, "  [%d/%d] ✓ Cached: %s\n", i+1, len(imageRefs), rawRef)
		}
	}

	if len(downloadErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d image(s) failed to download:\n", len(downloadErrors))
		for _, err := range downloadErrors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return fmt.Errorf("failed to download %d image(s)", len(downloadErrors))
	}

	return nil
}
