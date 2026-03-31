// Designed by Jerry Tse
package main

import (
	"encoding/json"
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

	sotaVersion, modules := extractAPKModules(updateRes)
	if len(modules) == 0 {
		fmt.Println("No available SOTA Update")
		os.Exit(0)
	}

	descRes, err := sota.FetchDescription(modules, sotaVersion, cfg)
	if err != nil {
		fmt.Printf("[!] Description failed: %v\n", err)
		os.Exit(1)
	}

	printChangelog(sotaVersion, descRes)
}

func extractAPKModules(res map[string]interface{}) (string, []map[string]interface{}) {
	sotaVersion := "Unknown"
	if sota, ok := res["sota"].(map[string]interface{}); ok {
		sotaVersion, _ = sota["sotaVersion"].(string)
	}

	moduleMap, _ := res["moduleMap"].(map[string]interface{})
	apkModules, _ := moduleMap["apk"].([]interface{})

	var modules []map[string]interface{}
	for _, m := range apkModules {
		mod, _ := m.(map[string]interface{})
		modules = append(modules, map[string]interface{}{
			"moduleName":    mod["moduleName"],
			"moduleVersion": mod["moduleVersion"],
		})
		if sotaVersion == "Unknown" {
			sotaVersion, _ = mod["sotaVersion"].(string)
		}
	}
	return sotaVersion, modules
}

func printChangelog(sotaVersion string, data map[string]interface{}) {
	bodyStr, _ := data["body"].(string)
	var body map[string]interface{}
	if bodyStr != "" {
		json.Unmarshal([]byte(bodyStr), &body)
	} else {
		body = data
	}

	moduleMap, _ := body["moduleMap"].(map[string]interface{})
	apkModules, _ := moduleMap["apk"].([]interface{})

	if len(apkModules) == 0 {
		fmt.Println("Not found apk changelog")
		return
	}

	fmt.Printf("Get SOTA Changelog from %s\n\n", sotaVersion)

	for _, m := range apkModules {
		module, _ := m.(map[string]interface{})
		descStr, _ := module["description"].(string)
		var desc map[string]interface{}
		json.Unmarshal([]byte(descStr), &desc)

		title, _ := desc["title"].(string)
		content, _ := desc["content"].([]interface{})
		if len(content) == 0 {
			continue
		}

		fmt.Println(title)
		for _, c := range content {
			item, _ := c.(map[string]interface{})
			fmt.Println(item["data"])
		}
		fmt.Println()
	}
}
