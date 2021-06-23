package ffthumb

import "os/exec"

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
