// Designed by Jerry Tse
package main

import (
	"fmt"
	"os"
	"strings"

	"oplus-tracker/pkg/sota"

	"github.com/spf13/pflag"
)

func main() {
	var cfg sota.SotaConfig
	pflag.StringVar(&cfg.Brand, "brand", "", "Device brand")
	pflag.StringVar(&cfg.OTAVersion, "ota-version", "", "OTA version")
	pflag.StringVar(&cfg.ColorOS, "coloros", "", "ColorOS version")
	pflag.Parse()

	if cfg.Brand == "" || cfg.OTAVersion == "" || cfg.ColorOS == "" {
		fmt.Println("Error: All parameters --brand, --ota-version, --coloros are required")
		os.Exit(1)
	}

	cfg.Model = strings.Split(cfg.OTAVersion, "_")[0]
	cfg.RomVersion = "unknown"

	fmt.Printf("Device: %s\n", cfg.Model)
	fmt.Printf("OS: %s\n\n", strings.Replace(cfg.ColorOS, "ColorOS", "ColorOS ", 1))

	queryRes, _, err := sota.Query(cfg)
	if err != nil {
		fmt.Printf("[!] Query failed: %v\n", err)
		os.Exit(1)
	}

	updateRes, err := sota.Update(queryRes, cfg)
	if err != nil {
		fmt.Printf("[!] Update failed: %v\n", err)
		os.Exit(1)
	}

	displaySotaResult(updateRes)
}

func displaySotaResult(res map[string]interface{}) {
	sotaVersion := "Unknown"
	if sota, ok := res["sota"].(map[string]interface{}); ok {
		sotaVersion, _ = sota["sotaVersion"].(string)
	}

	fmt.Println("SOTA Apk Info:")
	fmt.Printf("\n· SOTA Version: %s\n\n", sotaVersion)

	moduleMap, _ := res["moduleMap"].(map[string]interface{})
	apkModules, _ := moduleMap["apk"].([]interface{})

	for i, m := range apkModules {
		apk, _ := m.(map[string]interface{})
		fmt.Printf("• Apk Name: %v\n", apk["moduleName"])
		fmt.Printf("• Apk Version: %v\n", apk["moduleVersion"])
		fmt.Printf("• Apk Hash: %v\n", apk["md5"])
		fmt.Printf("• Link: %v\n", apk["manualUrl"])
		if i < len(apkModules)-1 {
			fmt.Println()
		}
	}
}
