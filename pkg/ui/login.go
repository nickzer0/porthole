package ui

// LoginPage returns the HTML for the login page
func LoginPage() string {
	return `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Porthole — Login</title>
    <meta name="ngrok-skip-browser-warning" content="true">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Archivo:wght@500;600;700&family=IBM+Plex+Mono:wght@400;500&display=swap" rel="stylesheet">
    <style>
        :root{
            --bg:#f6f5f1; --surface:#ffffff; --ink:#1a1f1d; --muted:#6e756f;
            --line:#e3e1d8; --accent:#0b7261; --accent-dark:#095e50;
            --accent-soft:#e3efec; --danger:#c0452f;
        }
        *{box-sizing:border-box;}
        body{
            margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
            padding:24px; background:var(--bg); color:var(--ink);
            font-family:'Archivo', sans-serif;
        }
        .card{
            width:360px; max-width:100%;
            background:var(--surface); border:1px solid var(--line); border-radius:12px;
            padding:40px 36px 36px;
        }
        .mark{
            width:40px; height:40px; margin:0 auto 18px; border-radius:50%;
            border:7px solid var(--accent);
        }
        h1{ margin:0; text-align:center; font-size:22px; font-weight:700; letter-spacing:-.01em; }
        .tag{
            text-align:center; font:400 11px/1 'IBM Plex Mono', monospace;
            color:var(--muted); margin:8px 0 28px;
        }
        label{
            display:block; font-size:13px; font-weight:600; margin-bottom:8px;
        }
        input[type="password"]{
            width:100%; padding:11px 14px; margin-bottom:20px;
            background:var(--surface); color:var(--ink);
            border:1px solid var(--line); border-radius:8px;
            font-family:'IBM Plex Mono', monospace; font-size:14px;
            transition:border-color .15s;
        }
        input[type="password"]::placeholder{ color:#b3b0a6; }
        input[type="password"]:focus{
            outline:none; border-color:var(--accent);
            box-shadow:0 0 0 3px var(--accent-soft);
        }
        button{
            width:100%; padding:12px; cursor:pointer;
            font-family:'Archivo', sans-serif; font-weight:600; font-size:14px;
            color:#fff; background:var(--accent);
            border:1px solid var(--accent); border-radius:8px;
            transition:background .15s;
        }
        button:hover{ background:var(--accent-dark); }
        .error{
            min-height:18px; color:var(--danger); text-align:center;
            font:400 12px/1.4 'IBM Plex Mono', monospace; margin-bottom:12px;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="mark"></div>
        <h1>Porthole</h1>
        <div class="tag">file transfer</div>
        <form method="POST">
            <div class="error" id="error"></div>
            <label for="pw">Password</label>
            <input id="pw" type="password" name="password" placeholder="Enter password" required autofocus>
            <button type="submit">Sign in</button>
        </form>
    </div>
    <script>
        const urlParams = new URLSearchParams(window.location.search);
        if (urlParams.get('error') === '1') {
            document.getElementById('error').textContent = 'Incorrect password';
        }
    </script>
</body>
</html>
`
}
