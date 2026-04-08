// Designed by Jerry Tse
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"oplus-tracker/pkg/config"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

type Child struct {
	Title   string        `json:"title"`
	Content []interface{} `json:"content"`
}

type UpgInstDetail struct {
	Children []Child `json:"children"`
	Link     string  `json:"link"`
	Content  string  `json:"content"`
	Title    string  `json:"title"`
	Type     string  `json:"type"`
}

type BodyData struct {
	UpgInstDetail []UpgInstDetail `json:"upgInstDetail"`
}

type Response struct {
	ResponseCode int    `json:"responseCode"`
	ErrMsg       string `json:"errMsg"`
	Body         string `json:"body"`
}

func main() {
	var pre int
	pflag.IntVar(&pre, "pre", -1, "Get Testing Version / Testing devices changelogs (0/1)")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: changelog_query <OTA_VERSION> <REGION> [--pre 0/1]")
		os.Exit(1)
	}

	otaVersion := strings.ToUpper(args[0])
	region := strings.ToLower(args[1])

	if strings.Count(otaVersion, "_") != 2 {
		fmt.Printf("Error: OTA_Prefix '%s' must contain exactly two underscores.\n", otaVersion)
		os.Exit(1)
	}

	regCfg, ok := config.REGION_CONFIG[region]
	if !ok {
		fmt.Printf("Invalid region: %s\n", region)
		os.Exit(1)
	}

	host := regCfg.Host
	if region != "cn" && region != "cn_cmcc" && region != "eu" && region != "in" {
		base := config.REGION_CONFIG["sg_host"]
		host = base.Host
	}

	pureModel, adjustedOTA := processVersionPrefix(otaVersion, pre)
	fullVersion := adjustedOTA + "_197001010000"

	innerParams := map[string]interface{}{
		"mode":           0,
		"maskOtaVersion": fullVersion,
		"bigVersion":     0,
		"h5LinkVersion":  6,
	}
	paramsJSON, _ := json.Marshal(innerParams)

	client := resty.New()
	url := fmt.Sprintf("https://%s/descriptionInfo", host)

	fmt.Printf("\nQuerying update log for %s\n\n", fullVersion)

	resp, err := client.R().
		SetHeaders(map[string]string{
			"language":       regCfg.Language,
			"nvCarrier":      regCfg.CarrierID,
			"mode":           "manual",
			"osVersion":      "unknown",
			"maskOtaVersion": fullVersion,
			"otaVersion":     fullVersion,
			"model":          pureModel,
			"androidVersion": "unknown",
			"Content-Type":   "application/json",
			"User-Agent":     "okhttp/4.12.0",
		}).
		SetBody(map[string]interface{}{
			"params": string(paramsJSON),
		}).
		Post(url)

	if err != nil {
		fmt.Printf("❌ Network error: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode() != 200 {
		fmt.Printf("❌ HTTP error: %d\n", resp.StatusCode())
		os.Exit(1)
	}

	var apiResp Response
	if err := json.Unmarshal(resp.Body(), &apiResp); err != nil {
		fmt.Println("❌ Response is not valid JSON.")
		os.Exit(1)
	}

	if apiResp.ResponseCode == 500 && apiResp.ErrMsg == "no modify" {
		fmt.Println("No changelog in Server")
		os.Exit(0)
	}

	if apiResp.ResponseCode != 200 {
		fmt.Printf("❌ API returned error code: %d\n", apiResp.ResponseCode)
		os.Exit(1)
	}

	var body BodyData
	if err := json.Unmarshal([]byte(apiResp.Body), &body); err != nil {
		fmt.Println("❌ 'body' content is not valid JSON.")
		os.Exit(1)
	}

	formatOutput(body, region)
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

func extractURLFromLink(linkStr string) string {
	re := regexp.MustCompile(`href\s*=\s*"([^"]+)"`)
	match := re.FindStringSubmatch(linkStr)
	if len(match) > 1 {
		return match[1]
	}
	return strings.TrimSpace(linkStr)
}

func formatOutput(body BodyData, region string) {
	chinaRegions := map[string]bool{"cn": true, "cn_cmcc": true}
	useBullet := chinaRegions[region]
	firstPrinted := false

	for _, item := range body.UpgInstDetail {
		if len(item.Children) > 0 {
			if firstPrinted {
				fmt.Println()
			}
			for j, child := range item.Children {
				if j > 0 {
					fmt.Println()
				}
				if child.Title != "" {
					fmt.Println(child.Title)
				}
				for _, c := range child.Content {
					var text string
					switch v := c.(type) {
					case string:
						text = v
					case map[string]interface{}:
						text, _ = v["data"].(string)
					}
					if text != "" {
						if useBullet {
							fmt.Printf("· %s\n", text)
						} else {
							fmt.Println(text)
						}
					}
				}
			}
			firstPrinted = true
		} else if item.Link != "" {
			if firstPrinted {
				fmt.Println()
			}
			if item.Content != "" {
				fmt.Println(item.Content)
			}
			url := extractURLFromLink(item.Link)
			fmt.Println(url)
			firstPrinted = true
		} else if item.Type == "updateTips" {
			if firstPrinted {
				fmt.Println()
			}
			title := item.Title
			if title == "" {
				title = "Important Notes"
			}
			fmt.Println(title)
			if item.Content != "" {
				fmt.Println(item.Content)
			}
			firstPrinted = true
		}
	}
}
