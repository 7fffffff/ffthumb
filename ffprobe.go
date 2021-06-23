package ffthumb

import "os/exec"

type ffProbeStream struct {
	CodecType          string `json:"codec_type"`
	SampleAspectRatio  string `json:"sample_aspect_ratio"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
}

type ffProbe struct {
	Streams []ffProbeStream `json:"streams"`
}

func findFFProbe(path string) string {
	if path != "" {
		p, err := exec.LookPath(path)
		if err == nil {
			return p
		}
		return ""
	}
	p, err := exec.LookPath("ffprobe")
	if err == nil {
		return p
	}
	return ""
}