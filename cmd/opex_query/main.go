// Designed by Jerry Tse
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"oplus-tracker/pkg/config"
	"oplus-tracker/pkg/crypto"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

type OpexInfo struct {
	Index        int    `json:"index"`
	VersionName  string `json:"version_name"`
	BusinessCode string `json:"business_code"`
	ZipHash      string `json:"zip_hash"`
	AutoURL      string `json:"auto_url"`
}

func parseOSVersion(v string) string {
	v = strings.TrimSpace(v)
	re := regexp.MustCompile(`^(\d+)(?:\.(\d+)(?:\.(\d+))?)?$`)
	match := re.FindStringSubmatch(v)
	if match != nil {
		major := match[1]
		minor := "0"
		if match[2] != "" {
			minor = match[2]
		}
		patch := "0"
		if match[3] != "" {
			patch = match[3]
		}
		return fmt.Sprintf("ColorOS%s.%s.%s", major, minor, patch)
	}
	if strings.Contains(v, "ColorOS") {
		return v
	}
	return v
}

func parseBrand(b string) string {
	b = strings.ToLower(strings.TrimSpace(b))
	switch b {
	case "oppo":
		return "OPPO"
	case "oneplus":
		return "OnePlus"
	case "realme":
		return "Realme"
	default:
		fmt.Printf("\nError: Invalid brand '%s'. Supported: OPPO, OnePlus, Realme\n", b)
		os.Exit(1)
		return ""
	}
}

func main() {
	var info string
	pflag.StringVar(&info, "info", "", "System info: osVersion,brand (e.g. 16,oneplus)")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 1 || info == "" {
		fmt.Println("Usage: opex_query <FULL_OTA_VERSION> --info <OS_VERSION>,<BRAND>")
		fmt.Println("Example: opex_query PJZ110_11.C.84_1840_202601060309 --info 16,oneplus")
		os.Exit(1)
	}

	otaVersion := args[0]
	parts := strings.Split(info, ",")
	if len(parts) != 2 {
		fmt.Println("Error: --info must be in format 'osVersion,brand'")
		os.Exit(1)
	}

	osVersion := parseOSVersion(parts[0])
	brand := parseBrand(parts[1])
	androidVersion := "Android" + parts[0]
	model := strings.Split(otaVersion, "_")[0]

	fmt.Printf("Querying Opex updates\n")
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Brand: %s\n", brand)
	fmt.Printf("OS: %s\n", strings.Replace(osVersion, "ColorOS", "ColorOS ", 1))

	url := fmt.Sprintf("https://%s%s", config.OPEX_CONFIG["host"], config.OPEX_CONFIG["endpoint"])

	client := resty.New()
	client.SetTimeout(30 * time.Second)

	for attempt := 0; attempt < 10; attempt++ {
		aesKey := crypto.GenerateRandomBytes(32)
		iv := crypto.GenerateRandomBytes(16)
		deviceID := strings.ToLower(crypto.GenerateRandomString(64))

		aesKeyB64 := base64.StdEncoding.EncodeToString(aesKey)
		protectedKey, err := crypto.RSAEncryptOAEP([]byte(aesKeyB64), config.OPEX_PUBLIC_KEY)
		if err != nil {
			continue
		}

		expireTime := fmt.Sprintf("%d", time.Now().UnixNano()+1e9*60*60*24)
		pkMap := map[string]interface{}{
			"opex": map[string]string{
				"protectedKey":       protectedKey,
				"version":            expireTime,
				"negotiationVersion": config.OPEX_CONFIG["public_key_version"],
			},
		}
		pkJSON, _ := json.Marshal(pkMap)

		headers := map[string]string{
			"language":       config.OPEX_CONFIG["language"],
			"newLanguage":    config.OPEX_CONFIG["language"],
			"androidVersion": androidVersion,
			"nvCarrier":      config.OPEX_CONFIG["carrier_id"],
			"deviceId":       deviceID,
			"osVersion":      osVersion,
			"productName":    model,
			"brand":          brand,
			"queryMode":      "0",
			"version":        "1",
			"Content-Type":   "application/json; charset=utf-8",
			"User-Agent":     "okhttp/5.3.2",
			"protectedKey":   string(pkJSON),
		}

		rawPayload := map[string]interface{}{
			"mode":         "0",
			"time":         time.Now().UnixNano() / 1e6,
			"businessList": []string{},
			"otaVersion":   otaVersion,
		}

		payloadJSON, _ := json.Marshal(rawPayload)
		cipherText, _ := crypto.AESCTREncrypt(payloadJSON, aesKey, iv)

		requestData := map[string]string{
			"cipher": base64.StdEncoding.EncodeToString(cipherText),
			"iv":     base64.StdEncoding.EncodeToString(iv),
		}

		resp, err := client.R().
			SetHeaders(headers).
			SetBody(requestData).
			Post(url)

		if err != nil || resp.StatusCode() != 200 {
			if attempt < 9 {
				time.Sleep(time.Duration(2*(attempt+1)) * time.Second)
				continue
			}
			break
		}

		var respJSON map[string]interface{}
		if err := json.Unmarshal(resp.Body(), &respJSON); err != nil {
			fmt.Printf("Error parsing response JSON: %v\n", err)
			continue
		}

		// 调试日志：打印服务器返回的原始 JSON
		// fmt.Printf("Debug: Raw Response: %s\n", resp.String())

		code, ok := respJSON["responseCode"].(float64)
		if !ok {
			code, _ = respJSON["code"].(float64)
		}

		if code == 500 {
			continue
		}

		if code != 200 && code != 0 {
			msg := "Unknown Error"
			if m, ok := respJSON["message"].(string); ok {
				msg = m
			} else if e, ok := respJSON["error"].(string); ok {
				msg = e
			}
			fmt.Printf("\nAPI Error (Code %.0f): %s\n", code, msg)
			return
		}

		// 如果 code 是 200 或 0，但没有 cipher 字段，说明可能不是加密响应
		if respJSON["cipher"] == nil {
			processResult(respJSON)
			return
		}

		// Decrypt response
		cipherB64, _ := respJSON["cipher"].(string)
		ivB64, _ := respJSON["iv"].(string)
		cipherBytes, _ := base64.StdEncoding.DecodeString(cipherB64)
		ivBytes, _ := base64.StdEncoding.DecodeString(ivB64)

		decrypted, err := crypto.AESCTRDecrypt(cipherBytes, aesKey, ivBytes)
		if err != nil {
			continue
		}

		var body map[string]interface{}
		json.Unmarshal(decrypted, &body)

		processResult(body)
		return
	}
}

func processResult(body map[string]interface{}) {
	var opexPackages []interface{}
	verName := "N/A"

	data := body["data"]
	if list, ok := data.([]interface{}); ok {
		opexPackages = list
		verName, _ = body["opexVersionName"].(string)
	} else if m, ok := data.(map[string]interface{}); ok {
		if list, ok := m["opexPackage"].([]interface{}); ok {
			opexPackages = list
		}
		verName, _ = m["opexVersionName"].(string)
	}

	found := false
	for _, p := range opexPackages {
		pkg, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		pCode, _ := pkg["code"].(float64)
		if pCode == 200 {
			info, ok := pkg["info"].(map[string]interface{})
			if !ok {
				continue
			}
			if !found {
				fmt.Println("\nOpex Info:")
				found = true
			} else {
				fmt.Println()
			}
			fmt.Printf("• Link: %s\n", info["autoUrl"])
			fmt.Printf("• Zip Hash: %s\n", info["zipHash"])
			fmt.Printf("• Opex Codename: %s\n", pkg["businessCode"])
			if verName != "" {
				fmt.Printf("• Opex Version: %s\n", verName)
			}
		}
	}

	if !found {
		fmt.Println("\nNo Opex updates found.")
	}
}
