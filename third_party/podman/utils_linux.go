// Copyright 2025 The Podman Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: This file was copied from https://github.com/containers/podman/blob/fb7e99786e8b38f88179b2504f1b55bb5a629d91/cmd/podman/images/utils_linux.go
// Original source is licensed under Apache License 2.0
// MODIFIED: This file has been modified from the original to match package naming conventions.

package podman

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// SetupPipe for fixing https://github.com/containers/podman/issues/7017
// uses named pipe since containers/image EvalSymlinks fails with /dev/stdout
// the caller should use the returned function to clean up the pipeDir
func SetupPipe() (string, func() <-chan error, error) {
	errc := make(chan error)
	pipeDir, err := os.MkdirTemp(os.TempDir(), "pipeDir")
	if err != nil {
		return "", nil, err
	}
	pipePath := filepath.Join(pipeDir, "saveio")
	err = unix.Mkfifo(pipePath, 0o600)
	if err != nil {
		if e := os.RemoveAll(pipeDir); e != nil {
			logrus.Errorf("Removing named pipe: %q", e)
		}
		return "", nil, fmt.Errorf("creating named pipe: %w", err)
	}
	go func() {
		fpipe, err := os.Open(pipePath)
		if err != nil {
			errc <- err
			return
		}
		_, err = io.Copy(os.Stdout, fpipe)
		fpipe.Close()
		errc <- err
	}()
	return pipePath, func() <-chan error {
		if e := os.RemoveAll(pipeDir); e != nil {
			logrus.Errorf("Removing named pipe: %q", e)
		}
		return errc
	}, nil
}
