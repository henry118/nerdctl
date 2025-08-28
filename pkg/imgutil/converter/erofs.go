/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package converter

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/core/images/converter"
	"github.com/containerd/containerd/v2/core/images/converter/uncompress"
	"github.com/containerd/containerd/v2/pkg/archive/compression"
	"github.com/containerd/errdefs"
	"github.com/containerd/nerdctl/v2/pkg/api/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const MediaTypeImageLayerEROFS = "application/vnd.oci.image.layer.erofs"

// Cache to avoid duplicate conversions
var cache Cache

func ErofsLayerConvertFunc(options types.ImageConvertOptions) (converter.ConvertFunc, error) {
	return func(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (*ocispec.Descriptor, error) {
		if !images.IsLayerType(desc.MediaType) {
			// No conversion. No need to return an error here.
			return nil, nil
		}

		// Check if already converted
		conv, added := cache.Add(desc.Digest.String())
		if !added {
			select {
			case <-conv.ch:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return &conv.result, nil
		}

		// Do the conversion
		var err error

		// Read it
		ra, err := cs.ReaderAt(ctx, desc)
		if err != nil {
			return nil, err
		}
		defer ra.Close()
		sr := io.NewSectionReader(ra, 0, desc.Size)

		info, err := cs.Info(ctx, desc.Digest)
		if err != nil {
			return nil, err
		}

		var r io.Reader
		// If it is compressed, get a decompressed stream
		if !uncompress.IsUncompressedType(desc.MediaType) {
			decompStream, err := compression.DecompressStream(sr)
			if err != nil {
				return nil, err
			}
			defer decompStream.Close()
			r = decompStream
		} else {
			r = sr
		}

		ref := fmt.Sprintf("convert-erofs-from-%s", desc.Digest)
		w, err := content.OpenWriter(ctx, cs, content.WithRef(ref))
		if err != nil {
			return nil, err
		}
		defer w.Close()

		// Reset the writing position
		// Old writer possibly remains without aborted
		// (e.g. conversion interrupted by a signal)
		if err := w.Truncate(0); err != nil {
			return nil, err
		}

		// Convert to erofs
		ew, err := NewErofsWriteCloser(w)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(ew, r)
		if err != nil {
			return nil, err
		}
		if err := ew.Close(); err != nil {
			return nil, err
		}

		info.Labels["containerd.io/uncompressed"] = w.Digest().String()
		if err = w.Commit(ctx, 0, "", content.WithLabels(info.Labels)); err != nil && !errdefs.IsAlreadyExists(err) {
			return nil, err
		}

		newDesc := desc
		newDesc.Digest = w.Digest()
		newDesc.Size = ew.Size()
		newDesc.MediaType = MediaTypeImageLayerEROFS

		conv.result = newDesc
		close(conv.ch)

		return &newDesc, nil
	}, nil
}

type erofsWriteCloser struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	imgPath string
	w       io.Writer
	size    int64
}

func NewErofsWriteCloser(dest io.Writer) (*erofsWriteCloser, error) {
	imgFile, err := os.CreateTemp("", "erofs-image-*.img")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file for erofs image: %w", err)
	}
	imgPath := imgFile.Name()
	imgFile.Close()

	cmd := exec.CommandContext(context.TODO(), "mkfs.erofs", "--tar=f", "--aufs", "--quiet", "-zlz4hc", "-Enoinline_data", imgPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		os.Remove(imgPath)
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		os.Remove(imgPath)
		return nil, fmt.Errorf("failed to start mkfs.erofs: %w", err)
	}
	return &erofsWriteCloser{
		cmd:     cmd,
		stdin:   stdin,
		imgPath: imgPath,
		w:       dest,
	}, nil
}

func (w *erofsWriteCloser) Write(p []byte) (int, error) {
	if w.stdin == nil {
		return 0, fmt.Errorf("erofsWriteCloser: write on closed pipe")
	}
	return w.stdin.Write(p)
}

func (w *erofsWriteCloser) Close() error {
	defer os.Remove(w.imgPath)
	w.stdin.Close()
	if err := w.cmd.Wait(); err != nil {
		return fmt.Errorf("mkfs.erofs command failed: %w", err)
	}
	imgFile, err := os.Open(w.imgPath)
	if err != nil {
		return fmt.Errorf("failed to open erofs image %q: %w", w.imgPath, err)
	}
	defer imgFile.Close()
	if w.size, err = io.Copy(w.w, imgFile); err != nil {
		return fmt.Errorf("failed to copy erofs image %q to destination: %w", w.imgPath, err)
	}
	return nil
}

func (w *erofsWriteCloser) Size() int64 {
	return w.size
}

type LayerConv struct {
	result ocispec.Descriptor
	ch     chan struct{}
}

type Cache struct {
	layers map[string]*LayerConv
	mu     sync.Mutex
}

func (c *Cache) Add(dgst string) (*LayerConv, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.layers == nil {
		c.layers = make(map[string]*LayerConv)
	}
	if lc, ok := c.layers[dgst]; ok {
		return lc, false
	}
	lc := &LayerConv{
		ch: make(chan struct{}),
	}
	c.layers[dgst] = lc
	return lc, true
}
