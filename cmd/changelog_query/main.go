// Designed by Jerry Tse
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"oplus-tracker/pkg/config"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

func main() {
	var pre int
	pflag.IntVar(&pre, "pre", -1, "Get Testing Version / Testing devices changelogs (0/1)")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: changelog_query <OTA_VERSION> <REGION> [--pre 0/1]")
		os.Exit(1)
	}

	otaVersion := args[0]
	region := strings.ToLower(args[1])

	regCfg, ok := config.REGION_CONFIG[region]
	if !ok {
		fmt.Printf("Invalid region: %s\n", region)
		os.Exit(1)
	}

	// Use sg_host if not cn/eu/in
	if region != "cn" && region != "cn_cmcc" && region != "eu" && region != "in" {
		base := config.REGION_CONFIG["sg_host"]
		base.Language = regCfg.Language
		base.CarrierID = regCfg.CarrierID
		regCfg = base
	}

	pureModel, adjustedOTA := processVersionPrefix(otaVersion, pre)

	client := resty.New()
	url := fmt.Sprintf("https://%s/update/log", regCfg.Host)

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"otaVersion": adjustedOTA,
			"model":      pureModel,
			"language":   regCfg.Language,
		}).
		Get(url)

	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		os.Exit(1)
	}

	if !resp.IsSuccess() || !strings.Contains(resp.Header().Get("Content-Type"), "application/json") {
		fmt.Printf("Error: Server returned non-JSON response\nStatus Code: %d\nResponse: %s\n", resp.StatusCode(), resp.String())
		os.Exit(1)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &data); err != nil {
		fmt.Println("Invalid JSON response")
		os.Exit(1)
	}

	formatOutput(data, region)
}

func processVersionPrefix(orig string, pre int) (string, string) {
	parts := strings.SplitN(orig, "_", 2)
	modelPart := parts[0]
	rest := ""
	if len(parts) == 2 {
		rest = "_" + parts[1]
	}

	pureModel := strings.ReplaceAll(modelPart, "PRE", "")

	adjusted := orig
	if pre == 1 {
		if !strings.Contains(modelPart, "PRE") {
			adjusted = pureModel + "PRE" + rest
		}
	} else if pre == 0 {
		adjusted = pureModel + rest
	}

	return pureModel, adjusted
}

func formatOutput(data map[string]interface{}, region string) {
	upgInstDetail, _ := data["upgInstDetail"].([]interface{})
	if len(upgInstDetail) == 0 {
		fmt.Println("No update details found.")
		return
	}

	chinaRegions := map[string]bool{"cn": true, "cn_cmcc": true}
	useBullet := chinaRegions[region]

	for i, item := range upgInstDetail {
		detail, _ := item.(map[string]interface{})
		if children, ok := detail["children"].([]interface{}); ok {
			if i > 0 {
				fmt.Println()
			}
			title, _ := detail["title"].(string)
			fmt.Printf("%s\n", title)
			for _, c := range children {
				child, _ := c.(map[string]interface{})
				content, _ := child["content"].(string)
				if useBullet {
					fmt.Printf("· %s\n", content)
				} else {
					fmt.Printf("- %s\n", content)
				}
			}
		}
	}
}
