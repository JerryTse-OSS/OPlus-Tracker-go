// Designed by Jerry Tse
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

func main() {
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 3 {
		fmt.Println("Usage: realme_edl_query <VERSION_NAME> <REGION> <DATE>")
		fmt.Println("Example: realme_edl_query \"RMX3888_16.0.3.500(CN01)\" CN 202601241320")
		os.Exit(1)
	}

	versionName := args[0]
	region := strings.ToUpper(args[1])
	datePrefix := args[2]

	if len(datePrefix) != 12 {
		fmt.Printf("Error: Date length is %d, expected 12 characters.\n", len(datePrefix))
		os.Exit(1)
	}

	var bucket, server string
	switch region {
	case "EU", "EUEX", "EEA", "TR":
		bucket, server = "GDPR", "rms01.realme.net"
	case "CN", "CH":
		bucket, server = "domestic", "rms11.realme.net"
	default:
		bucket, server = "export", "rms01.realme.net"
	}

	// VERSION_CLEAN = re.sub(r"^RMX\d+_", "", VERSION_NAME).replace("(", "").replace(")", "")
	re := regexp.MustCompile(`^RMX\d+_`)
	versionClean := re.ReplaceAllString(versionName, "")
	versionClean = strings.ReplaceAll(versionClean, "(", "")
	versionClean = strings.ReplaceAll(versionClean, ")", "")

	model := strings.Split(versionName, "_")[0]
	baseURL := fmt.Sprintf("https://%s/sw/%s%s_11_%s_%s", server, model, bucket, versionClean, datePrefix)

	fmt.Printf("Querying for %s\n\n", versionName)

	client := resty.New()
	client.SetTimeout(2 * time.Second)
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(5))

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	found := false
	var foundURL string
	var mu sync.Mutex

	// 并发检测 0000-9999
	const maxWorkers = 100
	sem := make(chan struct{}, maxWorkers)

	for i := 0; i <= 9999; i++ {
		select {
		case <-ctx.Done():
			goto END
		default:
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(suffix int) {
			defer wg.Done()
			defer func() { <-sem }()

			url := fmt.Sprintf("%s%04d.zip", baseURL, suffix)
			resp, err := client.R().SetContext(ctx).Head(url)
			if err == nil && resp.StatusCode() == http.StatusOK {
				mu.Lock()
				if !found {
					found = true
					foundURL = url
					cancel() // 找到后取消所有其他请求
				}
				mu.Unlock()
			}
		}(i)
	}

END:
	wg.Wait()

	if found {
		fmt.Println("Fetch Info:")
		fmt.Printf("• Link: %s\n", foundURL)
	} else {
		fmt.Println("Fetch Info:")
		fmt.Println("• Link: Not Found")
	}
}
