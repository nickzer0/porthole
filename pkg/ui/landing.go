package ui

// LandingPage returns a simple landing page that bypasses ngrok warning
func LandingPage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Porthole</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Archivo:wght@600;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root{
            --bg:#f6f5f1; --ink:#1a1f1d; --muted:#6e756f;
            --line:#e3e1d8; --accent:#0b7261;
        }
        *{box-sizing:border-box;}
        body{
            margin:0; height:100vh; display:flex; align-items:center; justify-content:center;
            background:var(--bg); color:var(--ink);
            font-family:'Archivo', sans-serif;
        }
        .loader{ text-align:center; }
        .spinner{
            width:44px; height:44px; margin:0 auto 20px; border-radius:50%;
            border:4px solid var(--line); border-top-color:var(--accent);
            animation:spin .9s linear infinite;
        }
        @keyframes spin{ to{ transform:rotate(360deg); } }
        h1{ margin:0 0 6px; font-size:18px; font-weight:700; letter-spacing:-.01em; }
        p{ margin:0; font:400 12px/1 'IBM Plex Mono', monospace; color:var(--muted); }
    </style>
    <script>
        // Immediately fetch login with the ngrok bypass header, then swap content in.
        fetch('/login', { headers: { 'ngrok-skip-browser-warning': 'true' } })
        .then(response => response.text())
        .then(html => {
            document.open();
            document.write(html);
            document.close();
            window.history.pushState({}, '', '/login');
        })
        .catch(err => {
            document.body.innerHTML = '<div class="loader"><h1>Porthole</h1><p>Connection failed</p></div>';
        });
    </script>
</head>
<body>
    <div class="loader">
        <div class="spinner"></div>
        <h1>Porthole</h1>
        <p>Connecting&hellip;</p>
    </div>
</body>
</html>
`
}
