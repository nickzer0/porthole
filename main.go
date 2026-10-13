package main

import (
	"Porthole/pkg/auth"
	"Porthole/pkg/cloudfront"
	"Porthole/pkg/config"
	"Porthole/pkg/files"
	"Porthole/pkg/handlers"
	"Porthole/pkg/ngrok"
	"Porthole/pkg/share"
	ws "Porthole/pkg/websocket"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	allowIPs := flag.String("allow-ips", "", "comma-separated IPs/CIDRs allowed to reach the app, in addition to TRUSTED_IPS")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg.AllowedIPs = *allowIPs

	// Display the password for this session
	fmt.Printf("🔑 Password: %s\n\n", cfg.Password)

	// Build the IP allowlist before anything starts listening, so a malformed
	// entry fails the run rather than silently allowing everyone through.
	ipFilter, err := auth.NewIPFilter(cfg.AllowlistEntries(), cfg.CFOriginSecret)
	if err != nil {
		log.Fatal(err)
	}
	if ipFilter.Enabled() {
		fmt.Printf("🔒 IP allowlist: %s\n", strings.Join(cfg.AllowlistEntries(), ", "))
	} else {
		fmt.Println("⚠️  No IP allowlist set (set TRUSTED_IPS in .env or pass --allow-ips)")
	}

	// Initialize components
	sessionManager := auth.NewSessionManager(cfg.Password)
	fileManager := files.NewManager(cfg.DownloadsDir)
	if err := fileManager.EnsureDownloadsDir(); err != nil {
		log.Fatalf("Failed to create downloads directory %s: %v", cfg.DownloadsDir, err)
	}
	fmt.Printf("📁 Downloads dir: %s\n", cfg.DownloadsDir)
	hub := ws.NewHub()
	shareManager := share.NewManager()
	h := handlers.New(sessionManager, fileManager, hub, shareManager)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Variables to track cleanup
	var tunnel *ngrok.Tunnel
	var cfUpdater *cloudfront.Updater
	var cloudFrontURL string
	ctx := context.Background()

	// Context for health checker cancellation
	healthCheckCtx, healthCheckCancel := context.WithCancel(context.Background())

	// Start ngrok tunnel
	fmt.Println("Starting ngrok tunnel...")
	tunnel, err = ngrok.Start(cfg.Port, cfg.NgrokAuthToken)
	if err != nil {
		log.Printf("Warning: Failed to start ngrok: %v", err)
		log.Println("Continuing without ngrok...")
	} else {
		publicURL := tunnel.GetPublicURL()
		fmt.Printf("\n✓ ngrok tunnel established!\n")
		fmt.Printf("  Public URL: %s\n", publicURL)
		fmt.Printf("  Local URL:  http://localhost:%s\n", cfg.Port)

		// Setup CloudFront if AWS credentials are provided
		if cfg.HasAWSCredentials() {
			cfUpdater, err = cloudfront.NewUpdater(ctx, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, cfg.AWSRegion)
			if err != nil {
				log.Printf("Warning: Failed to create CloudFront updater: %v", err)
			} else {
				// Create or update CloudFront distribution
				if cfg.CloudFrontDistID == "" {
					// No distribution ID - create new distribution
					fmt.Println("\nCreating new CloudFront distribution...")
					distributionID, domainName, err := cfUpdater.CreateDistribution(ctx, publicURL, cfg.CFOriginSecret)
					if err != nil {
						log.Printf("Warning: Failed to create CloudFront distribution: %v", err)
					} else {
						cloudFrontURL = fmt.Sprintf("https://%s", domainName)
						ipFilter.RequireCloudFront()
						fmt.Printf("✓ CloudFront distribution created!\n")
						fmt.Printf("  CloudFront URL: %s\n", cloudFrontURL)
						fmt.Printf("  Distribution ID: %s\n", distributionID)
						// Save distribution ID to .env file
						if err := cfg.SaveDistributionID(distributionID); err != nil {
							log.Printf("Warning: Failed to save distribution ID to .env: %v", err)
						} else {
							fmt.Printf("  Distribution ID saved to .env file\n")
						}
						fmt.Println("  Note: Distribution deployment takes 15-20 minutes")
					}
				} else {
					// Distribution ID exists - update origin
					fmt.Println("\nUpdating CloudFront distribution...")
					domainName, err := cfUpdater.UpdateOrigin(ctx, cfg.CloudFrontDistID, publicURL, cfg.CFOriginSecret)
					if err != nil {
						log.Printf("Warning: Failed to update CloudFront: %v", err)
					} else {
						cloudFrontURL = fmt.Sprintf("https://%s", domainName)
						ipFilter.RequireCloudFront()
						fmt.Printf("✓ CloudFront distribution %s updated with new origin!\n", cfg.CloudFrontDistID)
						fmt.Printf("  CloudFront URL: %s\n", cloudFrontURL)
						fmt.Println("  Note: CloudFront changes may take up to 5 mins to propagate")
					}
				}

				if cloudFrontURL != "" {
					fmt.Println("\n🔒 CloudFront-only mode is active: the ngrok URL now returns 403")
					fmt.Println("   for everyone. Use the CloudFront URL above. Allow up to 5 mins")
					fmt.Println("   (15-20 on a brand new distribution) for this to take effect.")
					fmt.Printf("   Local access on http://localhost:%s is never blocked.\n", cfg.Port)
				}
			}
		}
		fmt.Println()
	}

	// Register routes. The IP filter wraps everything else so denied requests
	// short-circuit before auth or any handler work.
	http.HandleFunc("/", ipFilter.Restrict(h.Landing)) // Landing page doesn't need auth or headers (will be loaded by ngrok first)
	http.HandleFunc("/login", ipFilter.Restrict(auth.AddNgrokHeaders(h.Login)))
	http.HandleFunc("/app", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Main))))
	http.HandleFunc("/upload", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Upload))))
	http.HandleFunc("/ws-upload", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.WebSocketUpload))))
	http.HandleFunc("/ws-files", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.WebSocketFiles))))
	http.HandleFunc("/ws-download", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.WebSocketDownload))))
	http.HandleFunc("/ws-notepad", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.WebSocketNotepad))))
	http.HandleFunc("/ws-events", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.WebSocketEvents))))
	http.HandleFunc("/files", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Files))))
	http.HandleFunc("/download/", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Download))))
	http.HandleFunc("/delete/", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Delete))))
	http.HandleFunc("/share", ipFilter.Restrict(auth.AddNgrokHeaders(sessionManager.RequireAuth(h.Share))))
	// Anonymous share links: token-gated, no session required. Exempt from the
	// IP allowlist (recipients are off-network by design) but still blocked
	// from bypassing CloudFront.
	http.HandleFunc("/dl/", ipFilter.RestrictOriginOnly(auth.AddNgrokHeaders(h.SharedDownload)))
	http.HandleFunc("/ws-dl/", ipFilter.RestrictOriginOnly(auth.AddNgrokHeaders(h.WebSocketSharedDownload)))

	// Create HTTP server with graceful shutdown support
	server := &http.Server{
		Addr: ":" + cfg.Port,
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Start CloudFront health checker if URL is available
	if cloudFrontURL != "" {
		go func() {
			fmt.Println("\nWaiting for CloudFront distribution to be ready...")

			// Create HTTP client with 15 second timeout
			client := &http.Client{
				Timeout: 15 * time.Second,
			}

			// Helper function to check CloudFront status
			checkStatus := func() bool {
				// Create request with context so it can be cancelled immediately
				req, err := http.NewRequestWithContext(healthCheckCtx, "GET", cloudFrontURL, nil)
				if err != nil {
					fmt.Printf("Checking CloudFront distribution status... (error creating request: %v)\n", err)
					return false
				}

				resp, err := client.Do(req)
				if err != nil {
					// Check if it's a timeout error
					if os.IsTimeout(err) {
						fmt.Println("Checking CloudFront distribution status... (request timed out after 15s)")
					} else {
						fmt.Printf("Checking CloudFront distribution status... (error: %v)\n", err)
					}
					return false
				}
				defer resp.Body.Close()

				if resp.StatusCode == 200 {
					fmt.Println("✓ CloudFront distribution is live!")
					return true
				}

				fmt.Printf("Checking CloudFront distribution status... (HTTP %d)\n", resp.StatusCode)
				return false
			}

			// Check immediately (don't wait 15 seconds)
			if checkStatus() {
				return
			}

			// Continue checking every 15 seconds until live or cancelled
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if checkStatus() {
						return
					}
				case <-healthCheckCtx.Done():
					// Context cancelled (shutdown signal received)
					return
				}
			}
		}()
	}

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n\n🛑 Received shutdown signal (Ctrl+C)")

	// Cancel health checker
	healthCheckCancel()

	// Shutdown HTTP server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Stop ngrok tunnel with timeout
	if tunnel != nil {
		fmt.Println("Stopping ngrok tunnel...")
		done := make(chan bool, 1)
		go func() {
			tunnel.Stop()
			done <- true
		}()

		select {
		case <-done:
			// Tunnel stopped successfully
		case <-time.After(2 * time.Second):
			fmt.Println("Warning: ngrok tunnel shutdown timed out after 2s (forcing exit)")
		}
	}

	fmt.Println("\n✓ Shutdown complete")
}
