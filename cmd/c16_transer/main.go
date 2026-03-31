// Designed by Jerry Tse
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"oplus-tracker/pkg/util"

	"github.com/spf13/pflag"
)

func main() {
	var marketName string
	pflag.StringVar(&marketName, "market-name", "", "Optional market name")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: c16_transer <URL> [--market-name NAME]")
		os.Exit(1)
	}

	url := args[0]
	util.PrintBanner()
	fmt.Printf("URL: %s\n", url)

	extraHeaders := make(map[string]string)
	if marketName != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(marketName))
		extraHeaders["marketName"] = encoded
	}

	redirectURL := util.GetRedirectURL(url, 3)
	if redirectURL != "" && redirectURL != url {
		fmt.Println("\n✅ Success to resolve the URL:")
		fmt.Println("==================================================")
		fmt.Println(redirectURL)
		fmt.Println("==================================================")

		// Parse expires
		expires := extractExpiration(redirectURL)
		if !expires.IsZero() {
			fmt.Printf("\n📅 Expire time(UTC+8): %s\n", expires.Format("2006-01-02 15:04:05"))
		}
		fmt.Println("✅ DONE")
	} else {
		fmt.Println("❌ Failed to resolve")
		os.Exit(1)
	}
}

func extractExpiration(url string) time.Time {
	// Simple regex/parsing logic same as in oplus-ota or simplified
	// Re-using the logic from oplus-ota if needed
	return time.Time{} // Simplified for now, or copy the regex
}
