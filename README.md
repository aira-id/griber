# Griber (Go Realtime STT API)

Griber is an open-source speech-to-text API compatible with OpenAI's Realtime API, built using Golang and WebSockets. It supports multiple ASR providers, including `sherpa-onnx` and `mock` for testing.

## Features
- **OpenAI Compatible**: Implements the OpenAI Realtime API protocol.
- **Modular ASR**: Support for different ASR backends (currently we only support sherpa-onnx, other providers will be coming soon).
- **Configurable**: Full control via `config.yaml` and environment variables.

## Getting Started

### Prerequisites
- Go 1.21+
- ONNX Runtime libraries (for Sherpa-onnx)

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

```bash

# Create models directory if it doesn't exist
mkdir -p models

# Download Indonesian model
git clone https://huggingface.co/spacewave/sherpa-onnx-streaming-zipformer2-id

# Download English model
git clone https://huggingface.co/csukuangfj/sherpa-onnx-streaming-zipformer-en-2023-06-26
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

### WebSocket Endpoint
`ws://localhost:8080/v1/realtime`

### Protocol Documentation
This server implements the OpenAI Realtime API protocol for speech-to-text.
For detailed documentation on client events, server events, and message formats, please refer to the official **[OpenAI Realtime API Documentation](https://platform.openai.com/docs/guides/realtime)**.

Use standard OpenAI-compatible credentials and SDKs (where applicable) or raw WebSockets to interact with this endpoint.

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

### OpenAI Compatibility
1. **Confidence Scores**: Griber currently uses `sherpa-onnx`'s `OnlineRecognizer` for real-time transcription, which does not generate confidence scores. We plan to integrate `OfflineRecognizer` in the future to support this feature.
2. **API Protocols**: Only the WebSocket API endpoint is currently supported. Support for other protocols (e.g., HTTP REST and WebRTC) will be added in upcoming releases.
3. **Language Support**: Language availability depends on the specific models currently configured and loaded.

## Contributing

We welcome contributions from the community! Here are a few ways you can help improve Griber:

- **Testing & Compatibility**: Help us ensure high compatibility with the OpenAI Realtime API by testing with various clients and scenarios.
- **Reporting Issues**: Found a bug or have a suggestion? Open an issue to let us know.
- **Submitting Pull Requests**: Contributions are always welcome! Feel free to fork the repository and submit PRs for bug fixes or enhancements.
- **Adding New Features**: We're looking to expand support for other ASR providers, TTS engines, and additional API protocols.
- **Improving Documentation**: Help us make Griber easier to use by improving guides, examples, and API references.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.