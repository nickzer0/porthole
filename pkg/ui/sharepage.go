package ui

import (
	"fmt"
	"html"
)

// ShareDownloadPage returns the HTML for the anonymous download page served
// to browsers at /dl/<token>. The download itself runs over WebSocket
// (/ws-dl/<token>) with a direct-HTTP fallback (/dl/<token>?direct=1).
// The token is hex so it is safe to embed; the filename is escaped for
// display and the saved name comes from the server's metadata message.
func ShareDownloadPage(token, fileName string, fileSize int64) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Porthole — Download</title>
    <meta name="ngrok-skip-browser-warning" content="true">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Archivo:wght@500;600;700&family=IBM+Plex+Mono:wght@400;500;600&display=swap" rel="stylesheet">
    <style>
        :root{
            --bg:#f6f5f1; --surface:#ffffff; --ink:#1a1f1d; --muted:#6e756f;
            --line:#e3e1d8; --accent:#0b7261; --accent-dark:#095e50;
            --accent-soft:#e3efec; --danger:#c0452f;
        }
        *{box-sizing:border-box;}
        body{
            margin:0; min-height:100vh; background:var(--bg); color:var(--ink);
            font-family:'Archivo', sans-serif; font-size:14px; line-height:1.5;
            display:flex; align-items:center; justify-content:center; padding:24px;
        }
        .card{
            background:var(--surface); border:1px solid var(--line); border-radius:12px;
            padding:36px 32px; max-width:460px; width:100%%; text-align:center;
        }
        .brand{ display:flex; align-items:center; justify-content:center; gap:10px; margin-bottom:28px; }
        .mark{ width:20px; height:20px; flex:none; border-radius:50%%; border:4px solid var(--accent); }
        .wordmark{ font-size:15px; font-weight:700; letter-spacing:-.01em; }
        .file-name{
            font:600 14px/1.5 'IBM Plex Mono', monospace; color:var(--accent);
            word-break:break-all; margin-bottom:6px;
        }
        .file-meta{ font:400 12px/1.4 'IBM Plex Mono', monospace; color:var(--muted); margin-bottom:24px; }
        button{
            font-family:'Archivo', sans-serif; font-weight:600; font-size:13px; cursor:pointer;
            color:#fff; background:var(--accent); border:1px solid var(--accent); border-radius:8px;
            padding:10px 28px; transition:background .15s, border-color .15s;
        }
        button:hover{ background:var(--accent-dark); border-color:var(--accent-dark); }
        button:disabled{ opacity:.45; cursor:not-allowed; }
        .progress-container{ display:none; margin-top:24px; }
        .progress-bar{ height:8px; width:100%%; border-radius:999px; overflow:hidden; background:var(--line); }
        .progress-fill{
            height:100%%; width:0%%; transition:width .3s; border-radius:999px; background:var(--accent);
        }
        .status{ margin-top:12px; font:400 12px/1.5 'IBM Plex Mono', monospace; color:var(--muted); }
        .success{ color:var(--accent); }
        .error{ color:var(--danger); }
    </style>
</head>
<body>
    <div class="card">
        <div class="brand">
            <div class="mark"></div>
            <div class="wordmark">Porthole</div>
        </div>
        <div class="file-name">%s</div>
        <div class="file-meta">%s</div>
        <button id="downloadBtn" onclick="startDownload()">Download</button>
        <div class="progress-container" id="progressContainer">
            <div class="progress-bar">
                <div class="progress-fill" id="progressFill"></div>
            </div>
            <div class="status" id="status"></div>
        </div>
    </div>

    <script>
        const token = '%s';

        function setStatus(text, cls) {
            const status = document.getElementById('status');
            status.textContent = text;
            status.className = 'status' + (cls ? ' ' + cls : '');
        }

        function downloadWebSocket() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const ws = new WebSocket(protocol + '//' + window.location.host + '/ws-dl/' + token);

            const chunks = [];
            let totalChunks = 0;
            let fileName = 'download';

            return new Promise((resolve, reject) => {
                ws.onmessage = (event) => {
                    const msg = JSON.parse(event.data);

                    if (msg.type === 'metadata') {
                        totalChunks = msg.totalChunks;
                        fileName = msg.fileName;
                    } else if (msg.type === 'chunk') {
                        const binaryString = atob(msg.data);
                        const bytes = new Uint8Array(binaryString.length);
                        for (let i = 0; i < binaryString.length; i++) {
                            bytes[i] = binaryString.charCodeAt(i);
                        }
                        chunks.push(bytes);

                        const percent = totalChunks > 0
                            ? Math.min(100, Math.round(((msg.chunkNumber + 1) / totalChunks) * 100)) : 0;
                        document.getElementById('progressFill').style.width = percent + '%%';
                        setStatus('Downloading… ' + percent + '%%');
                    } else if (msg.type === 'complete') {
                        ws.close();

                        const blob = new Blob(chunks);
                        const url = window.URL.createObjectURL(blob);
                        const a = document.createElement('a');
                        a.href = url;
                        a.download = fileName;
                        document.body.appendChild(a);
                        a.click();
                        document.body.removeChild(a);
                        window.URL.revokeObjectURL(url);

                        resolve();
                    } else if (msg.type === 'error') {
                        ws.close();
                        reject(new Error(msg.error));
                    }
                };

                ws.onerror = () => reject(new Error('connection failed'));
            });
        }

        async function startDownload() {
            const btn = document.getElementById('downloadBtn');
            btn.disabled = true;
            document.getElementById('progressContainer').style.display = 'block';
            document.getElementById('progressFill').style.width = '0%%';
            setStatus('Connecting…');

            try {
                await downloadWebSocket();
                document.getElementById('progressFill').style.width = '100%%';
                setStatus('Download complete', 'success');
            } catch (error) {
                console.warn('WebSocket download failed, falling back to direct download:', error);
                setStatus('Downloading…');
                window.location.href = '/dl/' + token + '?direct=1';
                setStatus('Download started', 'success');
            }

            btn.disabled = false;
        }
    </script>
</body>
</html>
`, html.EscapeString(fileName), formatBytes(fileSize), token)
}

// formatBytes renders a byte count in human-readable form
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
