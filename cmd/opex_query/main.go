// Designed by Jerry Tse
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"oplus-tracker/pkg/util"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

func main() {
	var info string
	pflag.StringVar(&info, "info", "", "OS version and brand (e.g. 16,oneplus)")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: opex_query <FULL_OTA_VERSION> --info <OS_VERSION>,<BRAND>")
		os.Exit(1)
	}

	otaVersion := args[0]
	if info == "" {
		fmt.Println("Error: --info is required")
		os.Exit(1)
	}

	parts := strings.Split(info, ",")
	if len(parts) != 2 {
		fmt.Println("Error: --info format should be <OS_VERSION>,<BRAND>")
		os.Exit(1)
	}
	osVersion := parts[0]
	brand := parts[1]

	model := strings.Split(otaVersion, "_")[0]

	client := resty.New()
	url := "https://component-ota-cn.allawntech.com/opex/query"

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"otaVersion": otaVersion,
			"model":      model,
			"osVersion":  osVersion,
			"brand":      brand,
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
		fmt.Printf("Error: Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	code, _ := data["responseCode"].(float64)
	if int(code) == 200 {
		body, ok := data["body"].(map[string]interface{})
		if !ok || body == nil {
			fmt.Println("No Result (Empty body)")
			return
		}
		opexPackages, _ := body["opexPackage"].([]interface{})
		fmt.Println("Opex Info:")
		for i, p := range opexPackages {
			pkg, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			pCode, _ := pkg["code"].(float64)
			if int(pCode) == 200 {
				infoPkg, ok := pkg["info"].(map[string]interface{})
				if !ok {
					continue
				}
				autoURL, _ := infoPkg["autoUrl"].(string)
				fmt.Printf("• Link: %s\n", util.ReplaceGaussURL(autoURL))
				fmt.Printf("• Zip Hash: %s\n", infoPkg["zipHash"])
				fmt.Printf("• Opex Codename: %s\n", pkg["businessCode"])
				fmt.Printf("• Opex Version Name: %s\n", body["opexVersionName"])
				if i < len(opexPackages)-1 {
					fmt.Println()
				}
			}
		}
	} else {
		fmt.Println("No Result")
	}
}
