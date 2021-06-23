package ffthumb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// If the video is anamorphic (pixel aspect ratio doesn't match the display
// aspect ratio), we need to scale the thumbnail output appropriately. If
// no scaling is required, the returned filter is ""
func aspectFilter(ctx context.Context, ffprobePath, inputPath string) (filter string, err error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "error",
		"-print_format", "json",
		"-show_entries", "stream=codec_type,sample_aspect_ratio,display_aspect_ratio",
		inputPath,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return filter, fmt.Errorf("ffprobe: %v", strings.TrimSpace(stderr.String()))
		}
		return filter, fmt.Errorf("ffprobe: %w", err)
	}
	p := ffProbe{}
	err = json.Unmarshal(stdout.Bytes(), &p)
	if err != nil {
		return filter, err
	}
	for _, stream := range p.Streams {
		if stream.CodecType == "video" {
			if stream.SampleAspectRatio == "1:1" || stream.SampleAspectRatio == "0:1" || stream.DisplayAspectRatio == "0:1" {
				return "", nil
			}
			return "scale=iw*sar:ih", nil
		}
	}
	return "", nil
}
