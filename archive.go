package main

import (
	"context"
	"fmt"
	"os"

	"go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/docker/archive"
	"go.podman.io/image/v5/docker/reference"
	"go.podman.io/image/v5/oci/layout"
	"go.podman.io/image/v5/signature"
	"go.podman.io/image/v5/types"
	"golang.org/x/term"
	"solunio.com/image-creator/third_party/podman"
)

// createArchiveFromCache creates a docker archive from cached images
// The terminal detection and pipe setup pattern is based on Podman's save.go implementation:
// https://github.com/containers/podman/blob/fb7e99786e8b38f88179b2504f1b55bb5a629d91/cmd/podman/images/save.go#L110-L132
func createArchiveFromCache(ctx context.Context, imageRefs []string, outputFile string, cacheDir string, force bool, sysCtx *types.SystemContext, policyContext *signature.PolicyContext) error {
	fmt.Fprintln(os.Stderr, "\nStage 2: Creating docker archive from cache...")

	// Determine output destination
	var outputDest string
	var cleanupFunc func() <-chan error
	writeToStdout := (outputFile == "")

	if writeToStdout {
		// Refuse to write binary data to a terminal (like podman save does)
		if term.IsTerminal(int(os.Stdout.Fd())) {
			return fmt.Errorf("refusing to write archive to terminal. Use -o to specify an output file or redirect to a file/pipe")
		}

		// Setup named pipe for streaming to stdout
		var err error
		outputDest, cleanupFunc, err = podman.SetupPipe()
		if err != nil {
			return fmt.Errorf("failed to setup pipe: %w", err)
		}
		if cleanupFunc != nil {
			defer func() {
				errc := cleanupFunc()
				if writeErr := <-errc; writeErr != nil {
					fmt.Fprintf(os.Stderr, "Error writing to pipe: %v\n", writeErr)
				}
			}()
		}
	} else {
		outputDest = outputFile
		// Check if file exists and handle based on force flag
		if _, err := os.Stat(outputDest); err == nil {
			if !force {
				return fmt.Errorf("output file %s already exists. Use -f to overwrite", outputDest)
			}
			// Remove existing file if force is enabled
			if err := os.Remove(outputDest); err != nil {
				return fmt.Errorf("failed to remove existing archive file: %w", err)
			}
		}
	}

	writer, err := archive.NewWriter(sysCtx, outputDest)
	if err != nil {
		return fmt.Errorf("failed to create docker archive writer: %w", err)
	}

	for i, rawRef := range imageRefs {
		fmt.Fprintf(os.Stderr, "  [%d/%d] Archiving: %s\n", i+1, len(imageRefs), rawRef)

		// Read from OCI layout cache using the same tag we used during download
		cacheTag := fmt.Sprintf("img-%d", i)
		srcRef, err := layout.NewReference(cacheDir, cacheTag)
		if err != nil {
			return fmt.Errorf("failed to create OCI cache reference: %w", err)
		}

		// Parse the image reference to extract the tag properly
		named, err := reference.ParseNormalizedNamed(rawRef)
		if err != nil {
			return fmt.Errorf("failed to parse image reference %s: %w", rawRef, err)
		}

		// Ensure we have a tagged reference
		var tagged reference.NamedTagged
		if t, ok := named.(reference.NamedTagged); ok {
			tagged = t
		} else {
			// Add latest tag if no tag specified
			tagged, err = reference.WithTag(named, "latest")
			if err != nil {
				return fmt.Errorf("failed to add latest tag to %s: %w", rawRef, err)
			}
		}

		// Create a reference in the docker archive with proper tag
		destRef, err := writer.NewReference(tagged)
		if err != nil {
			return fmt.Errorf("failed to create docker archive reference for %s: %w", tagged.String(), err)
		}

		// Copy from cache to archive - this is local and fast
		_, err = copy.Image(
			ctx,
			policyContext,
			destRef,
			srcRef,
			&copy.Options{
				SourceCtx:      sysCtx,
				DestinationCtx: sysCtx,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to archive image %s: %w", rawRef, err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close archive writer: %w", err)
	}

	return nil
}
