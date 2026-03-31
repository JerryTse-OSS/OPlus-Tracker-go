// Designed by Jerry Tse
package util

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

func ReplaceGaussURL(url string) string {
	if url == "" || url == "N/A" {
		return url
	}
	return strings.Replace(
		url,
		"https://gauss-otacostauto-cn.allawnfs.com/",
		"https://gauss-componentotacostmanual-cn.allawnfs.com/",
		-1,
	)
}

func GetRedirectURL(url string, maxRetries int) string {
	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	client.SetHeader("User-Agent", "Mozilla/5.0 (Linux; Android 13; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Mobile Safari/537.36")
	client.SetHeader("Accept", "application/json,text/html,*/*")
	client.SetHeader("Accept-Language", "zh-CN,zh;q=0.9")
	client.SetHeader("userId", "oplus-ota|00000001")

	for i := 0; i < maxRetries; i++ {
		resp, err := client.R().Get(url)
		if err != nil {
			if strings.Contains(err.Error(), "auto redirect is disabled") {
				// This is actually what we want for some versions of resty
				if resp != nil && resp.StatusCode() == http.StatusFound {
					return resp.Header().Get("Location")
				}
			}
			time.Sleep(time.Duration(2*(i+1)) * time.Second)
			continue
		}
		if resp.StatusCode() == http.StatusFound {
			return resp.Header().Get("Location")
		}
		return url
	}
	return url
}

func FormatSize(sizeStr string) string {
	// Simple passthrough or format if needed
	return sizeStr
}

func PrintBanner() {
	fmt.Println("==================================================")
	fmt.Println("Copyright (C) 2025-2026 Jerry Tse")
	fmt.Println("==================================================")
}
