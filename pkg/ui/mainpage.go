package ui

// MainPage returns the HTML for the main upload/download page
func MainPage() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Porthole</title>
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
            color:var(--amber); background:var(--amber-soft);
        }
        .ws-indicator::before{
            content:''; position:absolute; left:12px; top:50%; transform:translateY(-50%);
            width:6px; height:6px; border-radius:50%; background:currentColor;
        }

        main{ max-width:680px; margin:0 auto; padding:40px 24px 56px; }
        section{ margin-bottom:44px; }
        .sec-head{ display:flex; align-items:center; justify-content:space-between; margin-bottom:14px; }
        .sec-label{ font-size:13px; font-weight:700; text-transform:uppercase; letter-spacing:.08em; }

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

        button{
            font-family:'Archivo', sans-serif; font-weight:600; font-size:13px; cursor:pointer;
            color:#fff; background:var(--accent); border:1px solid var(--accent); border-radius:8px;
            padding:10px 28px; transition:background .15s, border-color .15s, color .15s;
        }
        button:hover{ background:var(--accent-dark); border-color:var(--accent-dark); }
        button:disabled{ opacity:.45; cursor:not-allowed; }
        .download-btn, .refresh-btn, .delete-btn{
            background:var(--surface); color:var(--ink); border:1px solid var(--line);
            padding:7px 16px; font-size:12px;
        }
        .download-btn:hover, .refresh-btn:hover{
            background:var(--accent-soft); color:var(--accent); border-color:var(--accent);
        }
        .delete-btn:hover{ background:#f8e9e5; color:var(--danger); border-color:var(--danger); }

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
        <span class="ws-indicator">HTTP</span>
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
    </main>

    <footer>Porthole &middot; http</footer>

    <script>
        function formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
        }

        function formatTime(seconds) {
            if (seconds < 60) return Math.round(seconds) + 's';
            const minutes = Math.floor(seconds / 60);
            const secs = Math.round(seconds % 60);
            return minutes + 'm ' + secs + 's';
        }

        function formatDate(dateString) {
            const date = new Date(dateString);
            return date.toLocaleString();
        }

        async function loadFiles() {
            const filesList = document.getElementById('filesList');

            try {
                const response = await fetch('/files');
                if (!response.ok) {
                    throw new Error('Failed to load files');
                }

                const files = await response.json();

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

            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'delete-btn';
            deleteBtn.textContent = 'Delete';
            deleteBtn.addEventListener('click', () => deleteFile(file.name));

            item.append(glyph, info, downloadBtn, deleteBtn);
            return item;
        }

        function downloadFile(filename) {
            window.location.href = '/download/' + encodeURIComponent(filename);
        }

        async function deleteFile(filename) {
            if (!confirm('Delete "' + filename + '"? This cannot be undone.')) {
                return;
            }

            try {
                const response = await fetch('/delete/' + encodeURIComponent(filename), {
                    method: 'DELETE'
                });
                if (!response.ok) {
                    throw new Error(await response.text());
                }
                loadFiles();
            } catch (error) {
                console.error('Delete error:', error);
                alert('Error deleting file: ' + error.message);
            }
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

            const chunkSize = 5 * 1024 * 1024; // 5MB chunks
            const totalChunks = Math.ceil(file.size / chunkSize);
            const startTime = Date.now();
            let uploadedBytes = 0;

            try {
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
                        body: formData
                    });

                    if (!response.ok) {
                        throw new Error('Upload failed: ' + response.statusText);
                    }

                    uploadedBytes = end;
                    const percent = Math.round((uploadedBytes / file.size) * 100);
                    progressFill.style.width = percent + '%';
                    progressFill.textContent = percent + '%';

                    const elapsed = (Date.now() - startTime) / 1000;
                    const speed = uploadedBytes / elapsed;
                    const remaining = (file.size - uploadedBytes) / speed;

                    status.textContent = 'Uploading: ' + formatBytes(uploadedBytes) + ' / ' + formatBytes(file.size) +
                                        ' (' + formatBytes(speed) + '/s)' +
                                        (remaining > 0 ? ' - ' + formatTime(remaining) + ' remaining' : '');
                }

                progressFill.textContent = '100%';
                status.textContent = 'Upload complete! (' + file.name + ')';
                status.className = 'status success';

                // Reload files list
                loadFiles();

                // Reset file input
                fileInput.value = '';
                resetDropzone();

            } catch (error) {
                console.error('Upload error:', error);
                status.textContent = 'Error during upload: ' + error.message;
                status.className = 'status error';
            }

            uploadBtn.disabled = false;
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

        // Load files on page load
        loadFiles();
    </script>
</body>
</html>
`
}
