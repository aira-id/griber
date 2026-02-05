# Griber (Go Realtime STT API)

Griber is an open-source speech-to-text API compatible with OpenAI's Realtime API, built using Golang and WebSockets. It supports multiple ASR providers, including `sherpa-onnx` and `mock` for testing.

## Features
- **OpenAI Compatible**: Implements the OpenAI Realtime API (WebSocket) and Transcription API (HTTP).
- **Audio Decoding**: Automatic conversion of various audio formats (MP3, WAV, etc.) using `ffmpeg`.
- **Advanced ASR Features**: Support for word-level timestamps, temperature-controlled inference, and verbose JSON output.

## Getting Started

### Prerequisites
- Go 1.21+
- ONNX Runtime libraries
- **ffmpeg** (required for HTTP API audio decoding)

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/aira-id/griber.git
   cd griber
   ```
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Setup ASR models (see [Model Setup](#model-setup) below).

## Model Setup

Griber requires pre-trained ONNX models to perform speech-to-text. By default, it looks for models in the `models/` directory.

### 1. Download Models
Download the streaming Zipformer models from Hugging Face:

- **Indonesian (ID)**: [sherpa-onnx-streaming-zipformer2-id](https://huggingface.co/spacewave/sherpa-onnx-streaming-zipformer2-id)
- **English (EN)**: [sherpa-onnx-streaming-zipformer-en-2023-06-26](https://huggingface.co/csukuangfj/sherpa-onnx-streaming-zipformer-en-2023-06-26)
- **Whisper Small**: [sherpa-onnx-whisper-small](https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-whisper-small.tar.bz2)

```bash
# Create models directory if it doesn't exist
mkdir -p models
cd models

# Download Indonesian model (streaming)
git clone https://huggingface.co/spacewave/sherpa-onnx-streaming-zipformer2-id

# Download English model (streaming)
git clone https://huggingface.co/csukuangfj/sherpa-onnx-streaming-zipformer-en-2023-06-26

# Download Whisper small model (non-streaming)
curl -SL -O https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-whisper-small.tar.bz2
tar xvf sherpa-onnx-whisper-small.tar.bz2
rm sherpa-onnx-whisper-small.tar.bz2
```

### 2. Directory Structure
Your structure should look like this:

```text
griber/
├── models/
│   ├── sherpa-onnx-streaming-zipformer2-id/
│   │   ├── encoder-iter-100000-avg-15-chunk-32-left-256.onnx
│   │   ├── decoder-iter-100000-avg-15-chunk-32-left-256.onnx
│   │   ├── joiner-iter-100000-avg-15-chunk-32-left-256.onnx
│   │   └── tokens.txt
│   └── sherpa-onnx-streaming-zipformer-en-2023-06-26/
│       ├── encoder-epoch-99-avg-1-chunk-16-left-128.onnx
│       ├── decoder-epoch-99-avg-1-chunk-16-left-128.onnx
│       ├── joiner-epoch-99-avg-1-chunk-16-left-128.onnx
│       └── tokens.txt
└── config.yaml
```

### 3. Verify `config.yaml`
Ensure the file names in `config.yaml` match the files you downloaded.

### Running the Server
```bash
go run main.go
```
The server will start on port `8080` (default).

## Configuration

Griber uses `config.yaml` for main configuration. Environment variables can also be used for most settings.

## API Usage

### WebSocket API (Real-time)
`ws://localhost:8080/v1/realtime`

This endpoint implements the OpenAI Realtime API protocol for streaming speech-to-text.
- **Constraint**: Only streaming models can be used with this endpoint.

### HTTP API (Transcription)
`POST http://localhost:8080/v1/audio/transcriptions`

OpenAI-compatible transcription endpoint for batch processing.

**Request Parameters:**
- `file` (required): The audio file to transcribe.
- `model` (required): Model name (e.g., `sherpa-onnx-whisper-small`).
- `language`: ISO language code (e.g., `en`, `id`).
- `temperature`: Sampling temperature (0.0 to 1.0).
- `response_format`: `json` or `verbose_json` (for word-level timestamps).

**Example Usage:**
```bash
curl http://localhost:8080/v1/audio/transcriptions \
  -F "file=@audio.mp3" \
  -F "model=sherpa-onnx-whisper-small" \
  -F "response_format=verbose_json"
```

## Client Example

A simple web client is provided in the `client/` directory to demonstrate real-time transcription.

### Usage
1. Start the Griber server:
   ```bash
   go run main.go
   ```
2. Open `client/index.html` in your web browser.
3. Allow microphone access when prompted.
4. Click "Connect" to start the session and begin speaking.
5. You should see real-time transcription results.

*Note: Ensure the server is running on `localhost:8080` (default).*

## Limitations

1. **API Protocols**: We currently support WebSocket (Realtime) and HTTP (Transcription). WebRTC support is planned.
2. **Language Support**: Depends on the specific models configured in `config.yaml`.
3. **Hardware Acceleration**: Currently optimized for CPU; GPU support is in the roadmap.

## Contributing

We welcome contributions from the community! Here are a few ways you can help improve Griber:

- **Testing & Compatibility**: Help us ensure high compatibility with the OpenAI Realtime API by testing with various clients and scenarios.
- **Reporting Issues**: Found a bug or have a suggestion? Open an issue to let us know.
- **Submitting Pull Requests**: Contributions are always welcome! Feel free to fork the repository and submit PRs for bug fixes or enhancements.
- **Adding New Features**: We're looking to expand support for other ASR providers, TTS engines, and additional API protocols.
- **Improving Documentation**: Help us make Griber easier to use by improving guides, examples, and API references.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.