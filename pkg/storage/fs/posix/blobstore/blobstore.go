// Copyright 2018-2021 CERN
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package blobstore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/lookup"
	"github.com/cs3org/reva/v2/pkg/utils"
	"github.com/pkg/errors"
)

// Blobstore provides an interface to an filesystem based blobstore
type Blobstore struct {
	root string
}

// New returns a new Blobstore
func New(root string) (*Blobstore, error) {
	err := os.MkdirAll(root, 0700)
	if err != nil {
		return nil, err
	}

	return &Blobstore{
		root: root,
	}, nil
}

// Upload stores some data in the blobstore under the given key
func (bs *Blobstore) Upload(spaceID, blobID string, blobSize int64, source string) error {
	dest, err := bs.path(spaceID, blobID)
	if err != nil {
		return err
	}
	// ensure parent path exists
	if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
		return errors.Wrap(err, "Decomposedfs: oCIS blobstore: error creating parent folders for blob")
	}

	if err := os.Rename(source, dest); err == nil {
		return nil
	}

	// Rename failed, file needs to be copied.
	file, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "Decomposedfs: oCIS blobstore: Can not open source file to upload")
	}
	defer file.Close()

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "could not open blob '%s' for writing", dest)
	}

	w := bufio.NewWriter(f)
	_, err = w.ReadFrom(file)
	if err != nil {
		return errors.Wrapf(err, "could not write blob '%s'", dest)
	}

	return w.Flush()
}

// Download retrieves a blob from the blobstore for reading
func (bs *Blobstore) Download(spaceID, blobID string, blobSize int64) (io.ReadCloser, error) {
	dest, err := bs.path(spaceID, blobID)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(dest)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read blob '%s'", dest)
	}
	fi, err := file.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "could not stat blob '%s'", dest)
	}
	if fi.Size() != blobSize {
		return nil, fmt.Errorf("blob has unexpected size. %d bytes expected, got %d bytes", blobSize, fi.Size())
	}

	return file, nil
}

// Delete deletes a blob from the blobstore
func (bs *Blobstore) Delete(spaceID, blobID string) error {
	dest, err := bs.path(spaceID, blobID)
	if err != nil {
		return err
	}
	if err := utils.RemoveItem(dest); err != nil {
		return errors.Wrapf(err, "could not delete blob '%s'", dest)
	}
	return nil
}

func (bs *Blobstore) path(spaceID, blobID string) (string, error) {
	if spaceID == "" {
		return "", fmt.Errorf("blobstore: spaceID is empty")
	}
	if blobID == "" {
		return "", fmt.Errorf("blobstore: blobID is empty")
	}
	return filepath.Join(
		bs.root,
		filepath.Clean(filepath.Join(
			"/", "spaces", lookup.Pathify(spaceID, 1, 2), "blobs", lookup.Pathify(blobID, 4, 2)),
		),
	), nil
}
