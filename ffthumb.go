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

type Thumbnailer struct {
	Num         int    // number of candidate thumbnails
	FFmpegPath  string // path to ffmpeg
	FFprobePath string // path to ffprobe
}

// LookupExec fills in FFmpegPath or FFprobePath if either is empty, by
// calling exec.LookPath to find the binaries in PATH
func (p *Thumbnailer) LookupExec() {
	if p.FFmpegPath == "" {
		p.FFmpegPath = findFFMpeg(p.FFmpegPath)
	}
	if p.FFprobePath == "" {
		p.FFprobePath = findFFProbe(p.FFprobePath)
	}
}

// WriteThumbnail writes a png image to output, with the same dimensions
// as the input video.
//
// WriteThumbnail chooses a thumbnail from a video file by creating Num
// thumbnails, then choosing the largest (in terms of file size). This
// is based on the idea that the largest, least compressible thumbnail
// image is likely to contain something interesting to look at.
//
// FFmpegPath and FFprobePath must be set before calling WriteThumbnail.
func (p *Thumbnailer) WriteThumbnail(ctx context.Context, output io.Writer, inputPath string) error {
	if p.FFmpegPath == "" {
		return errors.New("ffthumb: missing path to ffmpeg")
	}
	if p.FFprobePath == "" {
		return errors.New("ffthumb: missing path to ffprobe")
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
	videoFilter, err := aspectFilter(ctx, p.FFprobePath, inputPath)
	if err != nil {
		return fmt.Errorf("ffthumb: %w", err)
	}
	ffmpegParams := []string{
		"-loglevel", "16",
		"-skip_frame", "nokey",
		"-i", inputPath,
	}
	if videoFilter != "" {
		ffmpegParams = append(ffmpegParams, "-vf", videoFilter)
	}
	ffmpegParams = append(ffmpegParams,
		"-frames:v", strconv.Itoa(candidates),
		"-vsync", "vfr",
		"-y",
		"%d.png",
	)
	cmd := exec.CommandContext(ctx, p.FFmpegPath, ffmpegParams...)
	cmd.Dir, err = ioutil.TempDir("", "ffthumb-")
	if err != nil {
		return fmt.Errorf("ffthumb: couldn't create temporary dir: %w", err)
	}
	defer os.RemoveAll(cmd.Dir)
	stderr := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("ffthumb: ffmpeg: %v", strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("ffthumb: ffmpeg: %w", err)
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
		return errors.New("ffthumb: couldn't select a thumbnail")
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
