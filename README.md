# ffthumb

## What does it do?
Given a video file, ffthumb chooses a thumbnail image by extracting the first
few keyframes and selecting the largest and least compressible file.

ffmpeg is used for keyframe extraction and is not provided.

ffprobe is used to determine if aspect ratio correction is required, and is
also not provided.

Documentation:
https://godoc.org/github.com/7fffffff/ffthumb

## Why not just use the first frame?
Sometimes the first frame makes for a poor thumbnail, because it's all one
color or something like that.