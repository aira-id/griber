/**
 * Gribe Realtime STT Web Client
 * Compatible with OpenAI Realtime API
 */

class GribeClient {
    constructor() {
        this.ws = null;
        this.audioContext = null;
        this.processor = null;
        this.stream = null;
        this.isConnected = false;
        this.isRecording = false;

        // UI Elements - Realtime
        this.wsUrlInput = document.getElementById('ws-url');
        this.apiKeyRealtimeInput = document.getElementById('api-key-realtime');
        this.modelSelectRealtime = document.getElementById('model-select-realtime');
        this.languageSelectRealtime = document.getElementById('language-select-realtime');
        this.connectBtn = document.getElementById('connect-btn');
        this.disconnectBtn = document.getElementById('disconnect-btn');
        this.micBtn = document.getElementById('mic-btn');
        this.statusDot = document.getElementById('status-dot');
        this.statusText = document.getElementById('status-text');
        this.sessionInfo = document.getElementById('session-info');
        this.transcriptionLog = document.getElementById('transcription-log');
        this.currentTranscription = document.getElementById('current-transcription');

        // UI Elements - Upload
        this.httpUrlInput = document.getElementById('http-url');
        this.apiKeyUploadInput = document.getElementById('api-key-upload');
        this.modelSelectUpload = document.getElementById('model-select-upload');
        this.languageSelectUpload = document.getElementById('language-select-upload');
        this.formatSelect = document.getElementById('format-select');
        this.temperatureInput = document.getElementById('temperature');
        this.fileInput = document.getElementById('file-input');
        this.dropZone = document.getElementById('drop-zone');
        this.fileNameDisplay = document.getElementById('file-name');
        this.transcribeBtn = document.getElementById('transcribe-btn');
        this.jsonResult = document.getElementById('json-result');
        this.uploadStatus = document.getElementById('upload-status');

        // General UI
        this.toastContainer = document.getElementById('toast-container');
        this.selectedFile = null;

        this.setupEventListeners();
        this.setupTabs();
        this.setupFileUpload();
    }

    // Toast notification system
    showToast(message, type = 'error', title = null, duration = 5000) {
        const icons = {
            error: '❌',
            warning: '⚠️',
            success: '✅',
            info: 'ℹ️'
        };

        const titles = {
            error: 'Error',
            warning: 'Warning',
            success: 'Success',
            info: 'Info'
        };

        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.innerHTML = `
            <span class="toast-icon">${icons[type]}</span>
            <div class="toast-content">
                <div class="toast-title">${title || titles[type]}</div>
                <div class="toast-message">${message}</div>
            </div>
            <button class="toast-close" onclick="this.parentElement.remove()">✕</button>
        `;

        this.toastContainer.appendChild(toast);

        // Auto-remove after duration
        if (duration > 0) {
            setTimeout(() => {
                if (toast.parentElement) {
                    toast.classList.add('toast-hiding');
                    setTimeout(() => toast.remove(), 300);
                }
            }, duration);
        }

        return toast;
    }

    setupEventListeners() {
        this.connectBtn.onclick = () => this.connect();
        this.disconnectBtn.onclick = () => this.disconnect();
        this.micBtn.onclick = () => this.toggleMicrophone();

        this.modelSelectRealtime.onchange = () => this.updateSession();
        this.languageSelectRealtime.onchange = () => this.updateSession();

        this.transcribeBtn.onclick = () => this.transcribeFile();
    }

    setupTabs() {
        const tabBtns = document.querySelectorAll('.tab-btn');
        const tabContents = document.querySelectorAll('.tab-content');

        tabBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const tabId = btn.getAttribute('data-tab');

                // Update buttons
                tabBtns.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');

                // Update content
                tabContents.forEach(content => {
                    content.classList.remove('active');
                    if (content.id === `${tabId}-section`) {
                        content.classList.add('active');
                    }
                });

                this.log(`Switched to ${tabId} tab`, 'info');
            });
        });
    }

    setupFileUpload() {
        this.dropZone.onclick = () => this.fileInput.click();

        this.fileInput.onchange = (e) => {
            if (e.target.files.length > 0) {
                this.handleFileSelect(e.target.files[0]);
            }
        };

        this.dropZone.ondragover = (e) => {
            e.preventDefault();
            this.dropZone.classList.add('drag-over');
        };

        this.dropZone.ondragleave = () => {
            this.dropZone.classList.remove('drag-over');
        };

        this.dropZone.ondrop = (e) => {
            e.preventDefault();
            this.dropZone.classList.remove('drag-over');
            if (e.dataTransfer.files.length > 0) {
                this.handleFileSelect(e.dataTransfer.files[0]);
            }
        };
    }

    handleFileSelect(file) {
        this.selectedFile = file;
        this.fileNameDisplay.textContent = file.name;
        this.log(`Selected file: ${file.name} (${(file.size / 1024 / 1024).toFixed(2)} MB)`, 'info');
    }

    async transcribeFile() {
        if (!this.selectedFile) {
            this.showToast('Please select an audio file first.', 'warning', 'No File Selected');
            return;
        }

        const url = this.httpUrlInput.value;
        const apiKey = this.apiKeyUploadInput.value;
        const model = this.modelSelectUpload.value;
        const language = this.languageSelectUpload.value;
        const format = this.formatSelect.value;
        const temperature = this.temperatureInput.value;

        this.setUploadLoading(true);
        this.log(`Starting transcription for ${this.selectedFile.name}...`, 'info');

        const formData = new FormData();
        formData.append('file', this.selectedFile);
        formData.append('model', model);
        formData.append('language', language);
        formData.append('response_format', format);
        formData.append('temperature', temperature);

        const headers = {};
        if (apiKey) {
            headers['Authorization'] = `Bearer ${apiKey}`;
        }

        try {
            const start = Date.now();
            const response = await fetch(url, {
                method: 'POST',
                body: formData,
                headers: headers
            });

            const duration = ((Date.now() - start) / 1000).toFixed(2);

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ error: { message: 'HTTP Error ' + response.status } }));
                throw new Error(errorData.error?.message || 'Upload failed');
            }

            const result = await response.json();
            this.displayResult(result);
            this.log(`Transcription completed in ${duration}s`, 'success');
            this.showToast('Transcription completed successfully', 'success', 'Done');

        } catch (error) {
            this.log(`Transcription failed: ${error.message}`, 'error');
            this.showToast(error.message, 'error', 'Transcription Error');
            this.uploadStatus.textContent = 'Error';
        } finally {
            this.setUploadLoading(false);
        }
    }

    setUploadLoading(loading) {
        this.transcribeBtn.disabled = loading;
        this.fileInput.disabled = loading;
        this.uploadStatus.textContent = loading ? 'Transcribing...' : 'Ready';

        if (loading) {
            this.jsonResult.textContent = 'Processing audio... please wait.';
            this.jsonResult.classList.add('text-secondary');
        }
    }

    displayResult(result) {
        this.jsonResult.classList.remove('text-secondary');
        this.jsonResult.textContent = JSON.stringify(result, null, 2);
    }

    log(message, type = 'status') {
        const item = document.createElement('div');
        item.className = `log-item ${type}`;

        const timestamp = document.createElement('span');
        timestamp.className = 'timestamp';
        timestamp.textContent = new Date().toLocaleTimeString();

        const content = document.createElement('div');
        content.textContent = message;

        item.appendChild(timestamp);
        item.appendChild(content);

        this.transcriptionLog.appendChild(item);
        this.transcriptionLog.scrollTop = this.transcriptionLog.scrollHeight;
    }

    updateStatus(status) {
        this.statusDot.className = 'status-dot';
        switch (status) {
            case 'disconnected':
                this.statusDot.classList.add('status-disconnected');
                this.statusText.textContent = 'Disconnected';
                this.connectBtn.disabled = false;
                this.disconnectBtn.disabled = true;
                this.micBtn.disabled = true;
                this.isConnected = false;
                break;
            case 'connecting':
                this.statusDot.classList.add('status-connecting');
                this.statusText.textContent = 'Connecting...';
                this.connectBtn.disabled = true;
                break;
            case 'connected':
                this.statusDot.classList.add('status-connected');
                this.statusText.textContent = 'Connected';
                this.connectBtn.disabled = true;
                this.disconnectBtn.disabled = false;
                this.micBtn.disabled = false;
                this.isConnected = true;
                break;
        }
    }

    async connect() {
        const url = this.wsUrlInput.value;
        const apiKey = this.apiKeyRealtimeInput.value;

        this.updateStatus('connecting');
        this.log(`Connecting to ${url}...`);

        try {
            this.ws = new WebSocket(url);

            this.ws.onopen = () => {
                this.updateStatus('connected');
                this.log('WebSocket Connected', 'status');

                // Send initial session configuration
                this.updateSession();
            };

            this.ws.onclose = () => {
                this.updateStatus('disconnected');
                this.log('WebSocket Disconnected', 'status');
                this.stopMicrophone(); // Ensure mic stops if WS closes
            };

            this.ws.onerror = (error) => {
                this.log('WebSocket Error', 'error');
                this.showToast('Failed to connect to server. Please check the URL and try again.', 'error', 'Connection Error');
                console.error(error);
            };

            this.ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                this.handleServerEvent(data);
            };

        } catch (error) {
            this.log(`Connection failed: ${error.message}`, 'error');
            this.showToast(`Connection failed: ${error.message}`, 'error', 'Connection Error');
            this.updateStatus('disconnected');
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }

    updateSession() {
        if (!this.isConnected) return;

        const model = this.modelSelectRealtime.value;
        const language = this.languageSelectRealtime.value;

        this.log(`Updating session: model=${model}, language=${language}`, 'status');

        // Show loading state
        this.setModelLoading(true);

        this.sendEvent({
            type: 'session.update',
            session: {
                audio: {
                    input: {
                        transcription: {
                            model: model,
                            language: language
                        }
                    }
                }
            }
        });
    }

    setModelLoading(loading) {
        this.modelSelectRealtime.disabled = loading;
        this.languageSelectRealtime.disabled = loading;
        this.micBtn.disabled = loading || !this.isConnected;

        if (loading) {
            this.statusText.textContent = 'Loading model...';
            this.statusDot.className = 'status-dot status-connecting';
        } else {
            this.statusText.textContent = 'Connected';
            this.statusDot.className = 'status-dot status-connected';
        }
    }

    sendEvent(event) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(event));
        }
    }

    handleServerEvent(event) {
        console.log('Server Event:', event.type, event);

        switch (event.type) {
            case 'session.created':
                this.sessionInfo.textContent = `Session: ${event.session.id}`;
                this.log(`Session created: ${event.session.id}`, 'status');
                break;

            case 'session.updated':
                this.setModelLoading(false);
                this.log('Session configuration updated', 'status');
                this.showToast('Model loaded successfully', 'success', 'Ready', 3000);
                break;

            case 'error':
                this.setModelLoading(false);
                this.handleError(event.error);
                break;

            case 'conversation.item.input_audio_transcription.failed':
                this.handleTranscriptionError(event.error);
                break;

            case 'conversation.item.input_audio_transcription.delta':
                this.updateCurrentTranscription(event.delta, true);
                break;

            case 'conversation.item.input_audio_transcription.completed':
                this.log(event.transcript, 'transcript');
                this.clearCurrentTranscription();
                break;

            case 'response.audio_transcript.delta':
                // For STT service, this might not be relevant unless it's full conversational
                this.updateCurrentTranscription(event.delta, true);
                break;

            case 'response.done':
                // Reset or final touch
                break;
        }
    }

    handleError(error) {
        const code = error.code || 'unknown_error';
        const message = error.message || 'An unknown error occurred';
        const type = error.type || 'error';

        // Log to console for debugging
        console.error('Server Error:', error);

        // Log to transcription log
        this.log(`Error [${code}]: ${message}`, 'error');

        // Show toast based on error type/code
        let toastTitle = 'Error';
        let toastType = 'error';

        switch (code) {
            case 'invalid_model':
                toastTitle = 'Invalid Model';
                break;
            case 'unsupported_language':
                toastTitle = 'Unsupported Language';
                break;
            case 'provider_not_configured':
                toastTitle = 'Provider Not Configured';
                toastType = 'warning';
                break;
            case 'provider_initialization_failed':
                toastTitle = 'Provider Initialization Failed';
                break;
            case 'configuration_unavailable':
                toastTitle = 'Configuration Unavailable';
                break;
            case 'missing_field':
                toastTitle = 'Missing Required Field';
                toastType = 'warning';
                break;
            case 'buffer_full':
                toastTitle = 'Audio Buffer Full';
                toastType = 'warning';
                break;
            default:
                toastTitle = this.formatErrorCode(code);
        }

        this.showToast(message, toastType, toastTitle);
    }

    handleTranscriptionError(error) {
        const code = error.code || 'transcription_failed';
        const message = error.message || 'Transcription failed';

        console.error('Transcription Error:', error);
        this.log(`Transcription Error: ${message}`, 'error');

        let toastTitle = 'Transcription Failed';
        if (code === 'provider_not_configured') {
            toastTitle = 'ASR Not Configured';
            this.showToast(
                'Please select a model and language first, then try again.',
                'warning',
                toastTitle
            );
        } else if (code === 'transcription_timeout') {
            toastTitle = 'Transcription Timeout';
            this.showToast(message, 'warning', toastTitle);
        } else {
            this.showToast(message, 'error', toastTitle);
        }
    }

    formatErrorCode(code) {
        // Convert snake_case to Title Case
        return code
            .split('_')
            .map(word => word.charAt(0).toUpperCase() + word.slice(1))
            .join(' ');
    }

    updateCurrentTranscription(text, isDelta = false) {
        if (isDelta) {
            if (this.currentTranscription.querySelector('.delta-text')) {
                const deltaSpan = this.currentTranscription.querySelector('.delta-text');
                deltaSpan.textContent += text;
            } else {
                this.currentTranscription.innerHTML = `<span class="delta-text">${text}</span>`;
            }
        } else {
            this.currentTranscription.textContent = text;
        }
    }

    clearCurrentTranscription() {
        this.currentTranscription.innerHTML = '<span class="text-secondary">Waiting for speech...</span>';
    }

    async toggleMicrophone() {
        if (this.isRecording) {
            this.stopMicrophone();
        } else {
            await this.startMicrophone();
        }
    }

    async startMicrophone() {
        try {
            this.stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    channelCount: 1,
                    sampleRate: 16000,
                    echoCancellation: true,
                    noiseSuppression: true
                }
            });

            this.audioContext = new (window.AudioContext || window.webkitAudioContext)({
                sampleRate: 16000
            });

            const source = this.audioContext.createMediaStreamSource(this.stream);

            // ScriptProcessorNode is deprecated but widely supported for this use case
            // Buffer size 4096 is standard
            this.processor = this.audioContext.createScriptProcessor(4096, 1, 1);

            source.connect(this.processor);
            this.processor.connect(this.audioContext.destination);

            this.processor.onaudioprocess = (e) => {
                if (!this.isRecording) return;

                const inputData = e.inputBuffer.getChannelData(0);
                const pcmData = this.float32ToInt16(inputData);
                const base64Audio = this.arrayBufferToBase64(pcmData.buffer);

                this.sendEvent({
                    type: 'input_audio_buffer.append',
                    audio: base64Audio
                });
            };

            this.isRecording = true;
            this.micBtn.classList.remove('btn-success');
            this.micBtn.classList.add('btn-danger');
            document.getElementById('mic-text').textContent = 'Stop Microphone';
            document.getElementById('mic-icon').textContent = '⏹️';
            this.log('Microphone started', 'status');

        } catch (error) {
            this.log(`Microphone error: ${error.message}`, 'error');
            this.showToast(`Microphone error: ${error.message}`, 'error', 'Microphone Error');
            console.error(error);
        }
    }

    stopMicrophone() {
        if (!this.isRecording) return;

        this.isRecording = false;

        // Commit the buffer
        this.sendEvent({ type: 'input_audio_buffer.commit' });

        if (this.processor) {
            this.processor.disconnect();
            this.processor = null;
        }

        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
            this.stream = null;
        }

        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }

        this.micBtn.classList.remove('btn-danger');
        this.micBtn.classList.add('btn-success');
        document.getElementById('mic-text').textContent = 'Start Microphone';
        document.getElementById('mic-icon').textContent = '🎤';
        this.log('Microphone stopped', 'status');
    }

    // Utility: Convert Float32Array to Int16Array (PCM)
    float32ToInt16(buffer) {
        let l = buffer.length;
        let buf = new Int16Array(l);
        while (l--) {
            let s = Math.max(-1, Math.min(1, buffer[l]));
            buf[l] = s < 0 ? s * 0x8000 : s * 0x7FFF;
        }
        return buf;
    }

    // Utility: ArrayBuffer to Base64
    arrayBufferToBase64(buffer) {
        let binary = '';
        const bytes = new Uint8Array(buffer);
        const len = bytes.byteLength;
        for (let i = 0; i < len; i++) {
            binary += String.fromCharCode(bytes[i]);
        }
        return window.btoa(binary);
    }
}

// Initialize client
window.onload = () => {
    window.gribe = new GribeClient();
};
