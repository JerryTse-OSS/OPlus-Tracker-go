// Designed by Jerry Tse
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

func main() {
	var model string
	pflag.StringVar(&model, "model", "unknown", "Custom model")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: iot_query <OTA_PREFIX> <REGION> [--model MODEL]")
		os.Exit(1)
	}

	otaPrefix := args[0]
	region := strings.ToLower(args[1])

	if region != "cn" {
		fmt.Println("iot_query only supports cn region")
		os.Exit(1)
	}

	client := resty.New()
	url := "https://iota.coloros.com/patch/ota/v2"

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"otaVersion": otaPrefix,
			"model":      model,
		}).
		Get(url)

	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(resp.String())
}
