package cloudfront

import (
	"Porthole/pkg/auth"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// Updater handles CloudFront distribution updates
type Updater struct {
	client *cloudfront.Client
}

// NewUpdater creates a new CloudFront updater
func NewUpdater(ctx context.Context, accessKey, secretKey, region string) (*Updater, error) {
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS credentials not set")
	}

	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	return &Updater{
		client: cloudfront.NewFromConfig(cfg),
	}, nil
}

// CreateDistribution creates a new CloudFront distribution with the ngrok URL as origin.
// originSecret is injected as a custom origin header so the app can distinguish
// CloudFront traffic from direct hits on the ngrok URL.
// Returns (distributionID, domainName, error)
func (u *Updater) CreateDistribution(ctx context.Context, ngrokURL, originSecret string) (string, string, error) {
	// Extract domain from ngrok URL
	ngrokDomain := strings.TrimPrefix(ngrokURL, "https://")
	ngrokDomain = strings.TrimPrefix(ngrokDomain, "http://")

	// Create distribution config
	input := &cloudfront.CreateDistributionInput{
		DistributionConfig: &types.DistributionConfig{
			CallerReference: aws.String(fmt.Sprintf("porthole-%d", time.Now().Unix())),
			Comment:         aws.String("Auto-created by Porthole"),
			Enabled:         aws.Bool(true),
			IsIPV6Enabled:   aws.Bool(false),

			// Origins configuration
			Origins: &types.Origins{
				Quantity: aws.Int32(1),
				Items: []types.Origin{
					{
						Id:         aws.String("ngrok-origin"),
						DomainName: aws.String(ngrokDomain),
						CustomOriginConfig: &types.CustomOriginConfig{
							HTTPPort:             aws.Int32(80),
							HTTPSPort:            aws.Int32(443),
							OriginProtocolPolicy: types.OriginProtocolPolicyHttpsOnly, // ngrok requires HTTPS
							OriginSslProtocols: &types.OriginSslProtocols{
								Items:    []types.SslProtocol{types.SslProtocolTLSv12},
								Quantity: aws.Int32(1),
							},
						},
						CustomHeaders: &types.CustomHeaders{
							Quantity: aws.Int32(2),
							Items: []types.OriginCustomHeader{
								{
									HeaderName:  aws.String("ngrok-skip-browser-warning"),
									HeaderValue: aws.String("true"),
								},
								{
									HeaderName:  aws.String(auth.OriginSecretHeader),
									HeaderValue: aws.String(originSecret),
								},
							},
						},
					},
				},
			},

			// Default cache behavior
			DefaultCacheBehavior: &types.DefaultCacheBehavior{
				TargetOriginId:       aws.String("ngrok-origin"),
				ViewerProtocolPolicy: types.ViewerProtocolPolicyAllowAll, // Allow both HTTP and HTTPS

				// Use managed cache policy: CachingDisabled
				CachePolicyId: aws.String("4135ea2d-6df8-44a3-9df3-4b5a84be39ad"),

				// Use custom origin request policy for proper cookie/header/websocket forwarding
				OriginRequestPolicyId: aws.String("5aaf314d-d6ef-46d2-bbc0-1ab12e6d890c"),

				AllowedMethods: &types.AllowedMethods{
					Quantity: aws.Int32(7),
					Items: []types.Method{
						types.MethodHead,
						types.MethodDelete,
						types.MethodPost,
						types.MethodGet,
						types.MethodOptions,
						types.MethodPut,
						types.MethodPatch,
					},
					CachedMethods: &types.CachedMethods{
						Quantity: aws.Int32(2),
						Items: []types.Method{
							types.MethodHead,
							types.MethodGet,
						},
					},
				},
				Compress: aws.Bool(false), // Disable compression for WebSocket support
			},
			// Geo restrictions
			Restrictions: &types.Restrictions{
				GeoRestriction: &types.GeoRestriction{
					RestrictionType: types.GeoRestrictionTypeNone,
					Quantity:        aws.Int32(0),
				},
			},
			// Viewer certificate
			ViewerCertificate: &types.ViewerCertificate{
				CloudFrontDefaultCertificate: aws.Bool(true),
			},
		},
	}

	// Create the distribution
	result, err := u.client.CreateDistribution(ctx, input)
	if err != nil {
		return "", "", fmt.Errorf("failed to create distribution: %w", err)
	}

	distributionID := *result.Distribution.Id
	domainName := *result.Distribution.DomainName
	log.Printf("✓ CloudFront distribution created: %s", distributionID)
	log.Printf("  CloudFront URL: https://%s", domainName)
	log.Printf("  Note: Distribution deployment may take 15-20 minutes")

	return distributionID, domainName, nil
}

// UpdateOrigin updates the CloudFront distribution origin with a new ngrok URL
// and refreshes the custom origin headers, including the origin secret.
// Returns the CloudFront domain name
func (u *Updater) UpdateOrigin(ctx context.Context, distributionID, ngrokURL, originSecret string) (string, error) {
	// Get current distribution config
	getResult, err := u.client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get distribution: %w", err)
	}

	distribution := getResult.Distribution
	distConfig := distribution.DistributionConfig
	domainName := *distribution.DomainName

	// Extract domain from ngrok URL (remove https://)
	ngrokDomain := strings.TrimPrefix(ngrokURL, "https://")
	ngrokDomain = strings.TrimPrefix(ngrokDomain, "http://")

	// Update the origin domain
	if len(distConfig.Origins.Items) > 0 {
		distConfig.Origins.Items[0].DomainName = aws.String(ngrokDomain)

		// Ensure custom origin config for WebSocket support
		if distConfig.Origins.Items[0].CustomOriginConfig == nil {
			distConfig.Origins.Items[0].CustomOriginConfig = &types.CustomOriginConfig{
				HTTPPort:             aws.Int32(80),
				HTTPSPort:            aws.Int32(443),
				OriginProtocolPolicy: types.OriginProtocolPolicyHttpsOnly, // ngrok requires HTTPS
				OriginSslProtocols: &types.OriginSslProtocols{
					Items:    []types.SslProtocol{types.SslProtocolTLSv12},
					Quantity: aws.Int32(1),
				},
			}
		} else {
			distConfig.Origins.Items[0].CustomOriginConfig.OriginProtocolPolicy = types.OriginProtocolPolicyHttpsOnly
		}

		// Refresh the custom origin headers
		setOriginHeader(&distConfig.Origins.Items[0], "ngrok-skip-browser-warning", "true")
		setOriginHeader(&distConfig.Origins.Items[0], auth.OriginSecretHeader, originSecret)
	} else {
		return "", fmt.Errorf("no origins found in distribution")
	}

	// Update the distribution
	_, err = u.client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionID),
		DistributionConfig: distConfig,
		IfMatch:            getResult.ETag,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update distribution: %w", err)
	}

	// Create invalidation to clear cache
	u.createInvalidation(ctx, distributionID)

	return domainName, nil
}

// setOriginHeader adds or updates a custom origin header. CloudFront sends
// these on every origin request and overwrites any same-named header from the
// viewer, which is what makes the origin secret trustworthy at the app.
func setOriginHeader(origin *types.Origin, name, value string) {
	if origin.CustomHeaders == nil {
		origin.CustomHeaders = &types.CustomHeaders{
			Quantity: aws.Int32(1),
			Items: []types.OriginCustomHeader{
				{
					HeaderName:  aws.String(name),
					HeaderValue: aws.String(value),
				},
			},
		}
		return
	}

	// Check if header already exists
	for i, header := range origin.CustomHeaders.Items {
		if *header.HeaderName == name {
			origin.CustomHeaders.Items[i].HeaderValue = aws.String(value)
			return
		}
	}

	origin.CustomHeaders.Items = append(
		origin.CustomHeaders.Items,
		types.OriginCustomHeader{
			HeaderName:  aws.String(name),
			HeaderValue: aws.String(value),
		},
	)
	origin.CustomHeaders.Quantity = aws.Int32(int32(len(origin.CustomHeaders.Items)))
}

// DisableDistribution disables a CloudFront distribution (deletion will need to be done manually)
func (u *Updater) DisableDistribution(ctx context.Context, distributionID string) error {
	// Get current distribution
	getResult, err := u.client.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return fmt.Errorf("failed to get distribution: %w", err)
	}

	// Check if already disabled
	distConfig := getResult.Distribution.DistributionConfig
	if !*distConfig.Enabled {
		log.Printf("Distribution %s is already disabled", distributionID)
		return nil
	}

	// Disable the distribution
	log.Printf("Disabling CloudFront distribution %s...", distributionID)
	distConfig.Enabled = aws.Bool(false)

	_, err = u.client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionID),
		DistributionConfig: distConfig,
		IfMatch:            getResult.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to disable distribution: %w", err)
	}

	log.Printf("✓ Distribution disable initiated")
	log.Printf("  The distribution will be disabled in 15-20 minutes")
	log.Printf("  Once disabled, delete it manually from AWS Console or it will continue to incur minimal charges")
	return nil
}

// createInvalidation creates a cache invalidation for the distribution
func (u *Updater) createInvalidation(ctx context.Context, distributionID string) {
	_, err := u.client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distributionID),
		InvalidationBatch: &types.InvalidationBatch{
			CallerReference: aws.String(fmt.Sprintf("ngrok-update-%d", time.Now().Unix())),
			Paths: &types.Paths{
				Quantity: aws.Int32(1),
				Items:    []string{"/*"},
			},
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to create invalidation: %v", err)
	}
}
