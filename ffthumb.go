// Package ffthumb provides a utility for finding an "interesting" thumbnail
// frame from a video file. FFmpeg (https://www.ffmpeg.org) is required.
package ffthumb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func findFFMpeg(path string) string {
	if path != "" {
		p, err := exec.LookPath(path)
		if err == nil {
			return p
		}
		return ""
	}
	p, err := exec.LookPath("ffmpeg")
	if err == nil {
		return p
	}
	return ""
}

// If FFmpegPath is blank, Thumbnailer will look for ffmpeg in PATH.
type Thumbnailer struct {
	Num        int    // number of candidate thumbnails
	FFmpegPath string // path to ffmpeg
}

// WriteThumbnail writes a png image to output, with the same dimensions
// as the input video.
//
// WriteThumbnail chooses a thumbnail from a video file by creating Num
// thumbnails, then choosing the largest (in terms of file size). This
// is based on the idea that the largest, least compressible thumbnail
// image is likely to contain something interesting to look at.
func (p *Thumbnailer) WriteThumbnail(ctx context.Context, output io.Writer, inputPath string) error {
	ffmpegPath := findFFMpeg(p.FFmpegPath)
	if ffmpegPath == "" {
		return errors.New("ffthumb: couldn't find ffmpeg")
	}
	var err error
	inputPath, err = filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't get absolute path of input file: %w", err)
	}
	candidates := p.Num
	if candidates < 1 {
		candidates = 1
	}
	cmd := exec.CommandContext(ctx,
		ffmpegPath,
		"-loglevel", "16",
		"-skip_frame", "nokey",
		"-i", inputPath,
		"-frames:v", strconv.Itoa(candidates),
		"-vsync", "vfr",
		"-y",
		"%d.png",
	)
	cmd.Dir, err = ioutil.TempDir("", "ffthumb-")
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't create temporary dir: %w", err)
	}
	defer os.RemoveAll(cmd.Dir)
	errBuf := bytes.NewBuffer(nil)
	cmd.Stderr = errBuf
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffthumb: ffmpeg error: %s: %w", strings.TrimSpace(errBuf.String()), err)
	}
	largestSize := int64(0)
	largestPath := ""
	err = filepath.Walk(cmd.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(info.Name()) != ".png" {
			return nil
		}
		if info.Size() > largestSize {
			largestSize = info.Size()
			largestPath = info.Name()
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't read thumbnails: %w", err)
	}
	if largestPath == "" {
		return errors.New("ffthumb: could not select a thumbnail")
	}
	selected, err := os.Open(filepath.Join(cmd.Dir, largestPath))
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't open thumbnail: %w", err)
	}
	_, err = io.Copy(output, selected)
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't write to output: %w", err)
	}
	return nil
}
