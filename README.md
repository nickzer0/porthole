# Porthole

A password-protected file upload and download server with automatic ngrok tunnel support.

## Features

- **WebSocket-based file uploads** with HTTP fallback
- Real-time upload progress tracking
- File listing and download interface
- Password protection with session management
- Automatic ngrok tunnel for internet access
- **Automatic CloudFront distribution creation & management** (bypasses ngrok browser warning)
- Modular, maintainable codebase

## Quick Start

### 1. Install ngrok

```bash
# macOS
brew install ngrok

# Or download from https://ngrok.com/download
```

### 2. Set up environment

Create a `.env` file (or copy from `.env.example`):

```bash
cp .env.example .env
# Edit .env and set your PASSWORD and NGROK_AUTHTOKEN
```

Your `.env` file should look like:
```
PASSWORD=your-secure-password
NGROK_AUTHTOKEN=your-ngrok-token
PORT=8081

# Optional: For automatic CloudFront updates
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key
AWS_REGION=us-east-1
CLOUDFRONT_DISTRIBUTION_ID=E1234567890ABC
```

### 3. Run the application

```bash
# Build and run
go build -o Porthole
./Porthole

# Or run directly
go run main.go
```

The application automatically loads environment variables from `.env` file.

### 4. Access the application

The application will display the public ngrok URL and local URL:

```
✓ ngrok tunnel established!
  Public URL: https://abc123.ngrok.io
  Local URL:  http://localhost:8081
```

Visit either URL and log in with your password. The ngrok URL automatically bypasses the browser warning page.

## Environment Variables

The application automatically loads variables from a `.env` file in the same directory.

- `PASSWORD` (required): Password for accessing the application
- `NGROK_AUTHTOKEN` (optional): Your ngrok authentication token for stable URLs
- `PORT` (optional): Server port (defaults to 8081)

You can also set these as traditional environment variables if preferred.

## Usage

1. **Login**: Enter your password to access the application
2. **Upload**: Select a file and click "Upload" to upload it
3. **Download**: Click "Download" on any file in the list to download it
4. **Share**: Share the ngrok public URL with anyone who needs access (they'll need the password)

## CloudFront Setup (Optional - Bypasses ngrok Warning)

### Automatic CloudFront Creation

The application can **automatically create and manage** a CloudFront distribution for you. This completely bypasses the ngrok browser warning page.

**Setup:**
1. Add AWS credentials to your `.env` file:
   ```bash
   AWS_ACCESS_KEY_ID=your-aws-access-key
   AWS_SECRET_ACCESS_KEY=your-aws-secret-key
   AWS_REGION=us-east-1
   ```

2. Run the application - it will automatically:
   - **First run**: Create a new CloudFront distribution (saves ID to `.env`)
   - **Subsequent runs**: Update the distribution with the new ngrok URL

3. Use the CloudFront URL instead of ngrok URL (shown in console output)
   - Format: `https://d123abc.cloudfront.net`
   - **No browser warning appears when using CloudFront URL**

**IAM Permissions Required:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudfront:CreateDistribution",
        "cloudfront:GetDistribution",
        "cloudfront:UpdateDistribution",
        "cloudfront:DeleteDistribution",
        "cloudfront:CreateInvalidation"
      ],
      "Resource": "*"
    }
  ]
}
```

**Note**: Initial CloudFront distribution deployment takes 15-20 minutes. Updates take 5-15 minutes.

### Automatic Cleanup on Shutdown

When you press **Ctrl+C** to stop the application, you'll be prompted to optionally disable the CloudFront distribution:

```
🛑 Received shutdown signal (Ctrl+C)
⚠️  CloudFront distribution E123ABC will be disabled.
⚠️  After disabling, you must manually delete it from AWS Console to avoid charges.
Continue with disable? (y/N):
```

- **Choose "y"**: Distribution disable is initiated (completes in 15-20 min), ID removed from `.env`, **app exits immediately**
- **Choose "n"**: Distribution is preserved for next startup (recommended if restarting soon)

**Important**: After disabling, manually delete the distribution from AWS Console to stop all charges. Disabled distributions still incur minimal storage costs.

### Custom Domain Setup

See [CLAUDE.md](CLAUDE.md#cloudfront-setup) for detailed instructions on adding a custom domain (e.g., `files.yourdomain.com`) to your CloudFront distribution.

## Security Notes

- All routes are password-protected except the login page
- Sessions last 24 hours
- Use HTTPS (via ngrok or CloudFront) in production
- Consider implementing rate limiting for production use

## Development

See [CLAUDE.md](CLAUDE.md) for detailed architecture and development information.
