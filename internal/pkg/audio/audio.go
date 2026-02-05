package audio

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Decode converts input audio data to 16kHz, 16-bit, Mono PCM raw audio
// suitable for Sherpa-ONNX. It uses ffmpeg for conversion.
func Decode(input []byte) ([]byte, error) {
	// check if ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w. Please install ffmpeg to support audio decoding", err)
	}

	// Prepare ffmpeg command
	// -i pipe:0       : Read from stdin
	// -f s16le        : Output format signed 16-bit little endian
	// -ar 16000       : Output sample rate 16000 Hz
	// -ac 1           : Output channels 1 (mono)
	// pipe:1          : Write to stdout
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "16000", "-ac", "1", "-v", "quiet", "pipe:1")

	// Set up stdin
	cmd.Stdin = bytes.NewReader(input)

	// Set up stdout/stderr
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg conversion failed: %w, stderr: %s", err, errBuffer.String())
	}

	return outBuffer.Bytes(), nil
}
