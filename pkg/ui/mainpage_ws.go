package ui

// MainPageWithWebSocket returns the HTML for the main upload/download page with WebSocket support
func MainPageWithWebSocket() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Porthole</title>
    <meta name="ngrok-skip-browser-warning" content="true">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Archivo:wght@500;600;700&family=IBM+Plex+Mono:wght@400;500;600&display=swap" rel="stylesheet">
    <style>
        :root{
            --bg:#f6f5f1; --surface:#ffffff; --ink:#1a1f1d; --muted:#6e756f;
            --line:#e3e1d8; --accent:#0b7261; --accent-dark:#095e50;
            --accent-soft:#e3efec; --danger:#c0452f;
            --amber:#9a6a14; --amber-soft:#f5ead3;
        }
        *{box-sizing:border-box;}
        body{
            margin:0; min-height:100vh; background:var(--bg); color:var(--ink);
            font-family:'Archivo', sans-serif; font-size:14px; line-height:1.5;
        }
        a{color:inherit;}

        /* ---- header ---- */
        header{
            position:sticky; top:0; z-index:10;
            display:flex; align-items:center; justify-content:space-between; gap:16px;
            padding:14px 24px;
            background:var(--surface); border-bottom:1px solid var(--line);
        }
        .brand{ display:flex; align-items:center; gap:12px; }
        .mark{ width:24px; height:24px; flex:none; border-radius:50%; border:5px solid var(--accent); }
        .wordmark{ font-size:17px; font-weight:700; letter-spacing:-.01em; }

        .ws-indicator{
            position:relative; font:500 11px/1 'IBM Plex Mono', monospace;
            padding:7px 12px 7px 26px; border-radius:999px;
        }
        .ws-indicator::before{
            content:''; position:absolute; left:12px; top:50%; transform:translateY(-50%);
            width:6px; height:6px; border-radius:50%; background:currentColor;
        }
        .ws-enabled{ color:var(--accent); background:var(--accent-soft); }
        .ws-disabled{ color:var(--amber); background:var(--amber-soft); }

        /* ---- layout ---- */
        main{ max-width:680px; margin:0 auto; padding:40px 24px 56px; }
        section{ margin-bottom:44px; }
        .sec-head{ display:flex; align-items:center; justify-content:space-between; margin-bottom:14px; }
        .sec-label{ font-size:13px; font-weight:700; text-transform:uppercase; letter-spacing:.08em; }

        /* ---- dropzone ---- */
        .dropzone{
            background:var(--surface); border:1.5px dashed #c9c6bb; border-radius:12px;
            padding:44px 24px; text-align:center; cursor:pointer;
            transition:border-color .15s, background .15s;
        }
        .dropzone:hover{ border-color:var(--accent); }
        .dropzone.drag{ border-color:var(--accent); background:var(--accent-soft); }
        .dropzone svg{ width:28px; height:28px; color:var(--accent); margin-bottom:10px; }
        .hint{ font-size:13px; color:var(--muted); }
        .hint b{ color:var(--ink); font-weight:600; }
        .filename{
            display:none; font:600 13px/1.5 'IBM Plex Mono', monospace; color:var(--accent);
            word-break:break-all; max-width:420px; margin:0 auto;
        }
        .dropzone.loaded .hint{ display:none; }
        .dropzone.loaded .filename{ display:block; }

        .transfer-controls{ text-align:center; margin-top:20px; }

        /* ---- buttons ---- */
        button{
            font-family:'Archivo', sans-serif; font-weight:600; font-size:13px; cursor:pointer;
            color:#fff; background:var(--accent); border:1px solid var(--accent); border-radius:8px;
            padding:10px 28px; transition:background .15s, border-color .15s, color .15s;
        }
        button:hover{ background:var(--accent-dark); border-color:var(--accent-dark); }
        button:disabled{ opacity:.45; cursor:not-allowed; }
        .download-btn, .refresh-btn, .clear-btn, .delete-btn, .share-btn{
            background:var(--surface); color:var(--ink); border:1px solid var(--line);
            padding:7px 16px; font-size:12px;
        }
        .download-btn:hover, .refresh-btn:hover, .share-btn:hover{
            background:var(--accent-soft); color:var(--accent); border-color:var(--accent);
        }
        .clear-btn:hover, .delete-btn:hover{ background:#f8e9e5; color:var(--danger); border-color:var(--danger); }

        /* ---- progress ---- */
        .progress-container{ display:none; margin-top:24px; }
        .progress-bar{
            height:8px; width:100%; border-radius:999px; overflow:hidden; background:var(--line);
        }
        .progress-fill{
            height:100%; width:0%; transition:width .3s; border-radius:999px;
            background:var(--accent);
            display:flex; align-items:center; justify-content:flex-end;
            color:#fff; font:600 8px/1 'IBM Plex Mono', monospace; padding-right:6px; white-space:nowrap;
        }
        .status{ margin-top:12px; font:400 12px/1.5 'IBM Plex Mono', monospace; color:var(--muted); text-align:center; }
        .success{ color:var(--accent); }
        .error{ color:var(--danger); }

        /* ---- files ---- */
        .files-list{
            background:var(--surface); border:1px solid var(--line); border-radius:12px; overflow:hidden;
        }
        .file-item{ display:flex; align-items:center; gap:14px; padding:14px 18px;
            border-bottom:1px solid var(--line); transition:background .15s; }
        .file-item:last-child{ border-bottom:none; }
        .file-item:hover{ background:#fafaf7; }
        .file-glyph{ width:30px; height:30px; flex:none; border-radius:50%; background:var(--accent-soft); }
        .file-info{ flex:1; min-width:0; }
        .file-name{ font-weight:600; font-size:14px;
            white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
        .file-meta{ margin-top:3px; font:400 11px/1.4 'IBM Plex Mono', monospace; color:var(--muted); }
        .empty-state{ padding:36px; text-align:center;
            font:400 12px/1.5 'IBM Plex Mono', monospace; color:var(--muted); }
        .empty-state.error{ color:var(--danger); }

        /* ---- notepad ---- */
        .notepad-textarea{
            width:100%; min-height:150px; padding:14px 16px; resize:vertical;
            background:var(--surface); color:var(--ink);
            border:1px solid var(--line); border-radius:12px;
            font-family:'IBM Plex Mono', monospace; font-size:13px; line-height:1.6;
            transition:border-color .15s;
        }
        .notepad-textarea::placeholder{ color:#b3b0a6; }
        .notepad-textarea:focus{ outline:none; border-color:var(--accent);
            box-shadow:0 0 0 3px var(--accent-soft); }
        .notepad-buttons{ margin-top:12px; display:flex; gap:10px; align-items:center; }
        .notepad-status{ font:400 12px/1 'IBM Plex Mono', monospace; color:var(--muted); }

        footer{ text-align:center; padding:0 24px 36px;
            font:400 11px/1 'IBM Plex Mono', monospace; color:#b3b0a6; }
    </style>
</head>
<body>
    <header>
        <div class="brand">
            <div class="mark"></div>
            <div class="wordmark">Porthole</div>
        </div>
        <span id="wsIndicator" class="ws-indicator ws-enabled">WebSocket</span>
    </header>

    <main>
        <section>
            <div class="sec-head">
                <div class="sec-label">Upload</div>
            </div>
            <div class="dropzone" id="porthole">
                <input type="file" id="fileInput" hidden />
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M12 16V4"></path>
                    <path d="m7 9 5-5 5 5"></path>
                    <path d="M5 20h14"></path>
                </svg>
                <div class="hint">Drop a file here, or <b>click to browse</b></div>
                <div class="filename" id="fileName"></div>
            </div>
            <div class="transfer-controls">
                <button id="uploadBtn" onclick="uploadFile()">Upload</button>
                <div class="progress-container" id="progressContainer">
                    <div class="progress-bar">
                        <div class="progress-fill" id="progressFill">0%</div>
                    </div>
                    <div class="status" id="status"></div>
                </div>
            </div>
        </section>

        <section>
            <div class="sec-head">
                <div class="sec-label">Files</div>
                <button class="refresh-btn" onclick="loadFiles()">Refresh</button>
            </div>
            <div class="files-list" id="filesList">
                <div class="empty-state">Loading files&hellip;</div>
            </div>
        </section>

        <section>
            <div class="sec-head">
                <div class="sec-label">Notepad</div>
            </div>
            <textarea id="notepadContent" class="notepad-textarea" placeholder="Shared notes..."></textarea>
            <div class="notepad-buttons">
                <button onclick="saveNotepad()">Save</button>
                <button class="clear-btn" onclick="clearNotepad()">Clear</button>
                <span id="notepadStatus" class="notepad-status"></span>
            </div>
        </section>
    </main>

    <footer>Porthole &middot; ws / http</footer>

    <script>
        let useWebSocket = true;

        function formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }

        function formatDate(dateString) {
            const date = new Date(dateString);
            return date.toLocaleString();
        }

        // Build a file row with DOM APIs so filenames are never interpolated
        // into HTML or inline handlers (quotes in a name must not break out).
        function renderFileItem(file) {
            const item = document.createElement('div');
            item.className = 'file-item';

            const glyph = document.createElement('div');
            glyph.className = 'file-glyph';

            const info = document.createElement('div');
            info.className = 'file-info';
            const name = document.createElement('div');
            name.className = 'file-name';
            name.textContent = file.name;
            const meta = document.createElement('div');
            meta.className = 'file-meta';
            meta.textContent = formatBytes(file.size) + ' · ' + formatDate(file.modTime);
            info.append(name, meta);

            const downloadBtn = document.createElement('button');
            downloadBtn.className = 'download-btn';
            downloadBtn.textContent = 'Download';
            downloadBtn.addEventListener('click', () => downloadFile(file.name));

            const shareBtn = document.createElement('button');
            shareBtn.className = 'share-btn';
            shareBtn.textContent = 'Share';
            shareBtn.addEventListener('click', () => shareFile(file.name));

            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'delete-btn';
            deleteBtn.textContent = 'Delete';
            deleteBtn.addEventListener('click', () => deleteFile(file.name));

            item.append(glyph, info, downloadBtn, shareBtn, deleteBtn);
            return item;
        }

        async function shareFile(filename) {
            const hoursInput = prompt('Link expiry (hours):', '24');
            if (hoursInput === null) return;

            const hours = parseFloat(hoursInput);
            if (!isFinite(hours) || hours <= 0) {
                alert('Expiry must be a positive number of hours.');
                return;
            }

            try {
                const response = await fetch('/share', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'ngrok-skip-browser-warning': 'true'
                    },
                    body: JSON.stringify({
                        fileName: filename,
                        expiryMinutes: Math.round(hours * 60)
                    })
                });
                if (!response.ok) {
                    throw new Error(await response.text());
                }
                const link = await response.json();

                let copied = false;
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    try {
                        await navigator.clipboard.writeText(link.url);
                        copied = true;
                    } catch (e) { /* fall through to prompt */ }
                }

                const expires = new Date(link.expiresAt).toLocaleString();
                if (copied) {
                    alert('Link copied to clipboard (expires ' + expires + '):\n\n' + link.url);
                } else {
                    prompt('Share link (expires ' + expires + '):', link.url);
                }
            } catch (error) {
                console.error('Share error:', error);
                alert('Error creating share link: ' + error.message);
            }
        }

        async function downloadFileWebSocket(filename) {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-files');

            const chunks = [];
            let totalChunks = 0;
            let fileName = filename;

            return new Promise((resolve, reject) => {
                ws.onopen = () => {
                    ws.send(JSON.stringify({
                        type: 'start',
                        fileName: filename
                    }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);

                    if (msg.type === 'metadata') {
                        totalChunks = msg.totalChunks;
                        fileName = msg.fileName;
                        console.log('Downloading:', fileName, 'Size:', msg.fileSize, 'bytes');
                    } else if (msg.type === 'chunk') {
                        // Decode base64 chunk
                        const binaryString = atob(msg.data);
                        const bytes = new Uint8Array(binaryString.length);
                        for (let i = 0; i < binaryString.length; i++) {
                            bytes[i] = binaryString.charCodeAt(i);
                        }
                        chunks.push(bytes);
                        console.log('Received chunk', msg.chunkNumber + 1, '/', totalChunks);
                    } else if (msg.type === 'complete') {
                        ws.close();

                        // Combine all chunks into a single blob
                        const blob = new Blob(chunks);

                        // Create download link and trigger download
                        const url = window.URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = fileName;
                        document.body.appendChild(a);
                        a.click();
                        document.body.removeChild(a);
                        window.URL.revokeObjectURL(url);

                        resolve(msg.message);
                    } else if (msg.type === 'error') {
                        ws.close();
                        reject(new Error(msg.error));
                    }
                };

                ws.onerror = (error) => {
                    reject(error);
                };
            });
        }

        async function downloadFileHTTP(filename) {
            window.location.href = '/download/' + encodeURIComponent(filename);
        }

        async function downloadFile(filename) {
            if (useWebSocket) {
                try {
                    await downloadFileWebSocket(filename);
                } catch (wsError) {
                    console.warn('WebSocket download failed, falling back to HTTP:', wsError);
                    downloadFileHTTP(filename);
                }
            } else {
                downloadFileHTTP(filename);
            }
        }

        async function loadFilesWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-files');

            return new Promise((resolve, reject) => {
                // Guarantee the promise always settles so loadFiles() can fall
                // back to HTTP if the socket connects but never replies (e.g.
                // when proxied through ngrok/CloudFront).
                let settled = false;
                const finish = (fn, arg) => {
                    if (settled) return;
                    settled = true;
                    clearTimeout(timer);
                    try { ws.close(); } catch (e) {}
                    fn(arg);
                };
                const timer = setTimeout(
                    () => finish(reject, new Error('file list timed out')), 4000);

                ws.onopen = () => {
                    ws.send(JSON.stringify({ type: 'list' }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);

                    if (msg.type === 'files') {
                        finish(resolve, msg.files);
                    } else if (msg.type === 'error') {
                        finish(reject, new Error(msg.error));
                    }
                };

                ws.onerror = () => finish(reject, new Error('file list socket error'));
                ws.onclose = () => finish(reject, new Error('file list socket closed'));
            });
        }

        async function loadFilesHTTP() {
            const response = await fetch('/files', {
                headers: {
                    'ngrok-skip-browser-warning': 'true'
                }
            });
            if (!response.ok) {
                throw new Error('Failed to load files');
            }
            return await response.json();
        }

        async function loadFiles() {
            const filesList = document.getElementById('filesList');

            try {
                let files;
                if (useWebSocket) {
                    try {
                        files = await loadFilesWebSocket();
                    } catch (wsError) {
                        console.warn('WebSocket file listing failed, falling back to HTTP:', wsError);
                        files = await loadFilesHTTP();
                    }
                } else {
                    files = await loadFilesHTTP();
                }

                if (files.length === 0) {
                    filesList.innerHTML = '<div class="empty-state">No files yet</div>';
                    return;
                }

                filesList.innerHTML = '';
                files.forEach(file => filesList.append(renderFileItem(file)));
            } catch (error) {
                console.error('Error loading files:', error);
                filesList.innerHTML = '<div class="empty-state error">Error loading files</div>';
            }
        }

        async function deleteFileWebSocket(filename) {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-files');

            return new Promise((resolve, reject) => {
                let settled = false;
                const finish = (fn, arg) => {
                    if (settled) return;
                    settled = true;
                    clearTimeout(timer);
                    try { ws.close(); } catch (e) {}
                    fn(arg);
                };
                const timer = setTimeout(
                    () => finish(reject, new Error('delete timed out')), 4000);

                ws.onopen = () => {
                    ws.send(JSON.stringify({ type: 'delete', fileName: filename }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);

                    if (msg.type === 'deleted') {
                        finish(resolve, msg.message);
                    } else if (msg.type === 'error') {
                        finish(reject, new Error(msg.error));
                    }
                };

                ws.onerror = () => finish(reject, new Error('delete socket error'));
                ws.onclose = () => finish(reject, new Error('delete socket closed'));
            });
        }

        async function deleteFileHTTP(filename) {
            const response = await fetch('/delete/' + encodeURIComponent(filename), {
                method: 'DELETE',
                headers: {
                    'ngrok-skip-browser-warning': 'true'
                }
            });
            if (!response.ok) {
                throw new Error(await response.text());
            }
        }

        async function deleteFile(filename) {
            if (!confirm('Delete "' + filename + '"? This cannot be undone.')) {
                return;
            }

            try {
                if (useWebSocket) {
                    try {
                        await deleteFileWebSocket(filename);
                    } catch (wsError) {
                        console.warn('WebSocket delete failed, falling back to HTTP:', wsError);
                        await deleteFileHTTP(filename);
                    }
                } else {
                    await deleteFileHTTP(filename);
                }
                loadFiles();
            } catch (error) {
                console.error('Delete error:', error);
                alert('Error deleting file: ' + error.message);
            }
        }

        async function uploadFileWebSocket(file) {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-upload');

            const chunkSize = 512 * 1024; // 512KB chunks
            let offset = 0;
            const totalChunks = Math.ceil(file.size / chunkSize);
            let chunkNumber = 0;
            const startTime = Date.now();

            return new Promise((resolve, reject) => {
                ws.onopen = () => {
                    ws.send(JSON.stringify({
                        type: 'start',
                        fileName: file.name,
                        fileSize: file.size
                    }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);

                    if (msg.type === 'ready') {
                        sendNextChunk();
                    } else if (msg.type === 'progress') {
                        const bytesUploaded = Math.min((msg.chunkNumber + 1) * chunkSize, file.size);
                        updateProgress(msg.chunkNumber, totalChunks, bytesUploaded, startTime, file.size);
                        // Send next chunk immediately after progress update
                        sendNextChunk();
                    } else if (msg.type === 'complete') {
                        ws.close();
                        resolve(msg.message);
                    } else if (msg.type === 'error') {
                        ws.close();
                        reject(new Error(msg.error));
                    }
                };

                ws.onerror = (error) => {
                    reject(error);
                };

                function sendNextChunk() {
                    if (offset >= file.size) {
                        ws.send(JSON.stringify({ type: 'complete' }));
                        return;
                    }

                    const chunk = file.slice(offset, offset + chunkSize);
                    const reader = new FileReader();

                    reader.onload = () => {
                        // Convert ArrayBuffer to base64 in a way that handles large chunks
                        const bytes = new Uint8Array(reader.result);
                        let binary = '';
                        const batchSize = 0x8000; // Process 32KB at a time to avoid call stack issues
                        for (let i = 0; i < bytes.length; i += batchSize) {
                            const chunkBytes = bytes.subarray(i, Math.min(i + batchSize, bytes.length));
                            binary += String.fromCharCode.apply(null, chunkBytes);
                        }
                        const base64 = btoa(binary);

                        ws.send(JSON.stringify({
                            type: 'chunk',
                            chunkNumber: chunkNumber,
                            totalChunks: totalChunks,
                            data: base64
                        }));

                        offset += chunkSize;
                        chunkNumber++;
                    };

                    reader.readAsArrayBuffer(chunk);
                }
            });
        }

        async function uploadFileHTTP(file) {
            const chunkSize = 5 * 1024 * 1024; // 5MB chunks
            const totalChunks = Math.ceil(file.size / chunkSize);
            const startTime = Date.now();
            let uploadedBytes = 0;

            for (let chunkNumber = 0; chunkNumber < totalChunks; chunkNumber++) {
                const start = chunkNumber * chunkSize;
                const end = Math.min(start + chunkSize, file.size);
                const chunk = file.slice(start, end);

                const formData = new FormData();
                formData.append('chunk', chunk);
                formData.append('fileName', file.name);
                formData.append('chunkNumber', chunkNumber);
                formData.append('totalChunks', totalChunks);

                const response = await fetch('/upload', {
                    method: 'POST',
                    body: formData,
                    headers: {
                        'ngrok-skip-browser-warning': 'true'
                    }
                });

                if (!response.ok) {
                    throw new Error('Upload failed: ' + response.statusText);
                }

                uploadedBytes = end;
                updateProgress(chunkNumber, totalChunks, uploadedBytes, startTime, file.size);
            }

            return 'Upload complete: ' + file.name;
        }

        function updateProgress(chunkNumber, totalChunks, uploadedBytes, startTime, fileSize) {
            const progressFill = document.getElementById('progressFill');
            const status = document.getElementById('status');

            const percent = Math.min(100, Math.round((uploadedBytes / fileSize) * 100));
            progressFill.style.width = percent + '%';
            progressFill.textContent = percent + '%';

            const elapsed = (Date.now() - startTime) / 1000;
            const speed = uploadedBytes / elapsed;
            const remaining = (fileSize - uploadedBytes) / speed;

            const formatTime = (seconds) => {
                if (seconds < 60) return Math.round(seconds) + 's';
                const minutes = Math.floor(seconds / 60);
                const secs = Math.round(seconds % 60);
                return minutes + 'm ' + secs + 's';
            };

            status.textContent = 'Uploading: ' + formatBytes(uploadedBytes) + ' / ' + formatBytes(fileSize) +
                                ' (' + formatBytes(speed) + '/s)' +
                                (remaining > 0 && remaining < Infinity ? ' - ' + formatTime(remaining) + ' remaining' : '');
        }

        async function uploadFile() {
            const fileInput = document.getElementById('fileInput');
            const file = fileInput.files[0];
            if (!file) {
                alert('Please select a file.');
                return;
            }

            const uploadBtn = document.getElementById('uploadBtn');
            const progressContainer = document.getElementById('progressContainer');
            const progressFill = document.getElementById('progressFill');
            const status = document.getElementById('status');

            uploadBtn.disabled = true;
            progressContainer.style.display = 'block';
            progressFill.style.width = '0%';
            progressFill.textContent = '0%';
            status.textContent = 'Starting upload...';
            status.className = 'status';

            try {
                let message;
                if (useWebSocket) {
                    try {
                        message = await uploadFileWebSocket(file);
                    } catch (wsError) {
                        console.warn('WebSocket failed, falling back to HTTP:', wsError);
                        useWebSocket = false;
                        document.getElementById('wsIndicator').className = 'ws-indicator ws-disabled';
                        document.getElementById('wsIndicator').textContent = 'HTTP';
                        message = await uploadFileHTTP(file);
                    }
                } else {
                    message = await uploadFileHTTP(file);
                }

                progressFill.style.width = '100%';
                progressFill.textContent = '100%';
                status.textContent = message;
                status.className = 'status success';

                // Refresh file list after a short delay to ensure file is written
                setTimeout(() => {
                    loadFiles();
                }, 500);
                fileInput.value = '';
                resetDropzone();

            } catch (error) {
                console.error('Upload error:', error);
                status.textContent = 'Error during upload: ' + error.message;
                status.className = 'status error';
            }

            uploadBtn.disabled = false;
        }

        // Notepad functions
        function showNotepadStatus(message, isError) {
            const status = document.getElementById('notepadStatus');
            status.textContent = message;
            status.style.color = isError ? '#c0452f' : '#0b7261';
            setTimeout(() => { status.textContent = ''; }, 3000);
        }

        async function loadNotepad() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-notepad');

            return new Promise((resolve, reject) => {
                ws.onopen = () => {
                    ws.send(JSON.stringify({ type: 'get' }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'content') {
                        document.getElementById('notepadContent').value = msg.content || '';
                        ws.close();
                        resolve();
                    } else if (msg.type === 'error') {
                        ws.close();
                        reject(new Error(msg.content));
                    }
                };

                ws.onerror = (error) => {
                    reject(error);
                };
            });
        }

        async function saveNotepad() {
            const content = document.getElementById('notepadContent').value;
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-notepad');

            return new Promise((resolve, reject) => {
                ws.onopen = () => {
                    ws.send(JSON.stringify({ type: 'set', content: content }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'saved') {
                        showNotepadStatus('Saved!', false);
                        ws.close();
                        resolve();
                    } else if (msg.type === 'error') {
                        showNotepadStatus('Error: ' + msg.content, true);
                        ws.close();
                        reject(new Error(msg.content));
                    }
                };

                ws.onerror = (error) => {
                    showNotepadStatus('Connection error', true);
                    reject(error);
                };
            });
        }

        async function clearNotepad() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-notepad');

            return new Promise((resolve, reject) => {
                ws.onopen = () => {
                    ws.send(JSON.stringify({ type: 'clear' }));
                };

                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'cleared') {
                        document.getElementById('notepadContent').value = '';
                        showNotepadStatus('Cleared', false);
                        ws.close();
                        resolve();
                    } else if (msg.type === 'error') {
                        showNotepadStatus('Error: ' + msg.content, true);
                        ws.close();
                        reject(new Error(msg.content));
                    }
                };

                ws.onerror = (error) => {
                    showNotepadStatus('Connection error', true);
                    reject(error);
                };
            });
        }

        // --- Dropzone: click-to-select, drag & drop, filename display ---
        function resetDropzone() {
            const dropzone = document.getElementById('porthole');
            document.getElementById('fileName').textContent = '';
            dropzone.classList.remove('loaded');
        }

        (function() {
            const dropzone = document.getElementById('porthole');
            const fileInput = document.getElementById('fileInput');
            const fileNameEl = document.getElementById('fileName');

            function showName() {
                if (fileInput.files && fileInput.files[0]) {
                    fileNameEl.textContent = fileInput.files[0].name;
                    dropzone.classList.add('loaded');
                } else {
                    resetDropzone();
                }
            }

            dropzone.addEventListener('click', () => fileInput.click());
            fileInput.addEventListener('change', showName);

            ['dragenter', 'dragover'].forEach(ev =>
                dropzone.addEventListener(ev, (e) => { e.preventDefault(); dropzone.classList.add('drag'); }));
            ['dragleave', 'drop'].forEach(ev =>
                dropzone.addEventListener(ev, (e) => { e.preventDefault(); dropzone.classList.remove('drag'); }));

            dropzone.addEventListener('drop', (e) => {
                if (e.dataTransfer && e.dataTransfer.files.length) {
                    fileInput.files = e.dataTransfer.files;
                    showName();
                }
            });
        })();

        // --- Live sync: persistent events channel ---
        // Keeps one connection open so the server can push updates (notepad
        // saves, new uploads) to every open page without a manual refresh.
        function connectEvents() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-events');

            ws.onmessage = (event) => {
                let msg;
                try { msg = JSON.parse(event.data); } catch (e) { return; }

                if (msg.type === 'notepad') {
                    const ta = document.getElementById('notepadContent');
                    // Don't clobber what someone is actively typing here.
                    if (document.activeElement !== ta) {
                        ta.value = msg.content || '';
                    }
                } else if (msg.type === 'files-changed') {
                    loadFiles();
                }
            };

            // Reconnect if the connection drops (server restart, sleep, etc.).
            ws.onclose = () => { setTimeout(connectEvents, 2000); };
            ws.onerror = () => { ws.close(); };
        }

        // Load files and notepad on page load
        loadFiles();
        loadNotepad().catch(err => console.error('Failed to load notepad:', err));
        connectEvents();
    </script>
</body>
</html>
`
}
