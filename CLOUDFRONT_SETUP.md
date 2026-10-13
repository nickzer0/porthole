# CloudFront Setup Guide

This guide walks you through creating a CloudFront distribution for your Porthole application.

## Prerequisites

1. AWS account
2. Domain name (optional, but recommended)
3. SSL certificate in AWS Certificate Manager (if using custom domain)

## Step 1: Run Porthole Once

First, run the application to get an initial ngrok URL:

```bash
./Porthole
```

Note the ngrok URL, for example: `https://abc123.ngrok-free.app`

## Step 2: Create CloudFront Distribution

### Using AWS Console

1. Go to CloudFront console: https://console.aws.amazon.com/cloudfront/v3/home
2. Click "Create distribution"

### Origin Settings

- **Origin Domain**: `abc123.ngrok-free.app` (your ngrok domain WITHOUT https://)
- **Protocol**: HTTPS only
- **HTTPS port**: 443
- **Minimum origin SSL protocol**: TLSv1.2

### Custom Headers

Add this custom header:
- **Header name**: `ngrok-skip-browser-warning`
- **Value**: `true`

### Default Cache Behavior

- **Viewer protocol policy**: Redirect HTTP to HTTPS
- **Allowed HTTP methods**: GET, HEAD, OPTIONS, PUT, POST, PATCH, DELETE
- **Cache policy**: CachingDisabled
- **Origin request policy**: AllViewer
- **Response headers policy**: (optional) CORS-with-preflight

### Distribution Settings

- **Price class**: Choose based on your needs (Use All Edge Locations for best performance)
- **Alternate domain name (CNAME)**: `files.yourdomain.com` (if using custom domain)
- **Custom SSL certificate**: Select your ACM certificate (if using custom domain)
- **Default root object**: (leave empty)

### Create the Distribution

Click "Create distribution" and wait 5-15 minutes for deployment.

Note your **Distribution ID** (e.g., `E1234567890ABC`)

## Step 3: Update DNS (if using custom domain)

In your DNS provider (Route 53, Cloudflare, etc.):

1. Create a CNAME record:
   - **Name**: `files` (or your subdomain)
   - **Type**: CNAME
   - **Value**: `d123abc.cloudfront.net` (your CloudFront domain)
   - **TTL**: 300

## Step 4: Set Up IAM User

### Create IAM Policy

1. Go to IAM console: https://console.aws.amazon.com/iam/
2. Create a new policy with this JSON:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudfront:GetDistribution",
        "cloudfront:UpdateDistribution",
        "cloudfront:CreateInvalidation"
      ],
      "Resource": "*"
    }
  ]
}
```

Save as `CloudFrontUpdatePolicy`

### Create IAM User

1. Create new IAM user (e.g., `cloudfront-updater`)
2. Attach the `CloudFrontUpdatePolicy`
3. Create access key
4. Note the **Access Key ID** and **Secret Access Key**

## Step 5: Configure Porthole

Add to your `.env` file:

```bash
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-east-1
CLOUDFRONT_DISTRIBUTION_ID=E1234567890ABC
```

## Step 6: Test Automatic Updates

Restart the application:

```bash
./Porthole
```

You should see:

```
✓ ngrok tunnel established!
  Public URL: https://xyz789.ngrok-free.app

Updating CloudFront distribution...
✓ CloudFront distribution E1234567890ABC updated with new origin!
  Note: CloudFront changes may take 5-15 minutes to propagate
```

## Verification

After 5-15 minutes:

1. Visit your CloudFront URL: `https://d123abc.cloudfront.net`
2. OR your custom domain: `https://files.yourdomain.com`
3. You should see the login page (no ngrok warning!)

## Troubleshooting

### "Failed to update CloudFront"

- Check AWS credentials are correct
- Verify IAM user has required permissions
- Ensure distribution ID is correct

### CloudFront shows old origin

- Wait 5-15 minutes for changes to propagate
- Check CloudFront distribution in AWS Console to verify origin was updated
- Create manual invalidation in CloudFront console if needed

### "Access Denied" errors

- Verify IAM policy is attached to user
- Check AWS region is set correctly (CloudFront is global but uses us-east-1)

### ngrok warning still appears

- Check custom header is set: `ngrok-skip-browser-warning: true`
- Wait for CloudFront cache to clear (or create invalidation)

## Cost Considerations

- **CloudFront**: Pay for data transfer and requests (free tier: 1TB/month for 12 months)
- **ngrok**: Free tier has bandwidth limits (check ngrok pricing)
- **Cache invalidations**: First 1,000/month free, $0.005 per path after

## Benefits of This Setup

1. **Stable URL**: Custom domain never changes, even when ngrok URL changes
2. **Professional**: Use your own domain instead of ngrok subdomain
3. **SSL/TLS**: Free SSL certificate via AWS Certificate Manager
4. **Global CDN**: CloudFront edge locations for faster access worldwide
5. **Automatic**: App handles ngrok URL changes on every restart
