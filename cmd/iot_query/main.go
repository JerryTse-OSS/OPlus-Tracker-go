// Designed by Jerry Tse
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

	"oplus-tracker/pkg/config"
	"oplus-tracker/pkg/crypto"
	"oplus-tracker/pkg/util"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

func getKey(keyPseudo string) []byte {
	index := int(keyPseudo[0] - '0')
	realKey := config.IOT_OLD_KEYS[index] + keyPseudo[4:12]
	return []byte(realKey)
}

func encryptECB(data string) (string, error) {
	keyPseudo := fmt.Sprintf("%d", rand.Intn(10)) + crypto.GenerateRandomString(14)
	keyReal := getKey(keyPseudo)

	ciphertext, err := crypto.AESECBEncrypt([]byte(data), keyReal)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(ciphertext) + keyPseudo, nil
}

func decryptECB(encryptedData string) (string, error) {
	if len(encryptedData) < 15 {
		return "", fmt.Errorf("invalid encrypted data length")
	}
	ciphertextB64 := encryptedData[:len(encryptedData)-15]
	keyPseudo := encryptedData[len(encryptedData)-15:]

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	keyReal := getKey(keyPseudo)
	plaintext, err := crypto.AESECBDecrypt(ciphertext, keyReal)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func queryIoTServer(otaVersion, model string) map[string]interface{} {
	lang := "zh-CN"
	parts := strings.Split(otaVersion, "_")
	romVersion := otaVersion
	if len(parts) >= 3 {
		romVersion = strings.Join(parts[:3], "_")
	}

	headers := map[string]string{
		"language":       lang,
		"newLanguage":    lang,
		"romVersion":     romVersion,
		"otaVersion":     otaVersion,
		"androidVersion": "unknown",
		"colorOSVersion": "unknown",
		"model":          model,
		"infVersion":     "1",
		"nvCarrier":      "10010111",
		"deviceId":       strings.Repeat("0", 64),
		"mode":           "client_auto",
		"version":        "1",
		"Accept":         "application/json",
		"Content-Type":   "application/json",
	}

	isRealme := "0"
	if strings.Contains(model, "RMX") {
		isRealme = "1"
	}

	body := map[string]interface{}{
		"language":    lang,
		"romVersion":  romVersion,
		"otaVersion":  otaVersion,
		"model":       model,
		"productName": model,
		"imei":        strings.Repeat("0", 15),
		"mode":        "0",
		"deviceId":    strings.Repeat("0", 64),
		"version":     "2",
		"type":        "1",
		"isRealme":    isRealme,
		"time":        fmt.Sprintf("%d", time.Now().UnixNano()/1e6),
	}

	bodyJSON, _ := json.Marshal(body)
	encryptedParams, err := encryptECB(string(bodyJSON))
	if err != nil {
		return nil
	}

	client := resty.New()
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(map[string]interface{}{"params": encryptedParams}).
		Post(config.IOT_SPECIAL_SERVER_CN)

	if err != nil || resp.StatusCode() != 200 {
		return nil
	}

	var respJSON map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &respJSON); err != nil {
		return nil
	}

	if code, ok := respJSON["responseCode"].(float64); ok && code != 200 {
		return nil
	}

	resps, _ := respJSON["resps"].(string)
	if resps == "" {
		return nil
	}

	decryptedStr, err := decryptECB(resps)
	if err != nil {
		return nil
	}

	var decryptedJSON map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedStr), &decryptedJSON); err != nil {
		return nil
	}

	if decryptedJSON["checkFailReason"] != nil {
		return nil
	}

	return decryptedJSON
}

func displayIoTResult(res map[string]interface{}) {
	downURL, _ := res["down_url"].(string)
	changelog, _ := res["description"].(string)
	patch, _ := res["googlePatchLevel"].(string)
	version, _ := res["new_version"].(string)

	fmt.Println("Fetch Info:")
	fmt.Printf("• Link: %s\n", util.ReplaceGaussURL(downURL))
	fmt.Printf("• Changelog: %s\n", util.ReplaceGaussURL(changelog))
	fmt.Printf("• Security Patch: %s\n", strings.ReplaceAll(patch, "0", "N/A"))
	fmt.Printf("• Version: %s\n", version)
	fmt.Printf("• Ota Version: %s\n", version)
}

func main() {
	var model string
	pflag.StringVar(&model, "model", "", "Custom model override")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: iot_query <OTA_PREFIX> cn [--model MODEL]")
		os.Exit(1)
	}

	otaInput := strings.ToUpper(args[0])
	region := strings.ToLower(args[1])

	if region != "cn" {
		fmt.Println("Error: iot_query only supports cn region")
		os.Exit(1)
	}

	rand.Seed(time.Now().UnixNano())

	isSimple := !regexp.MustCompile(`_\d{2}\.[A-Z]`).MatchString(otaInput) && strings.Count(otaInput, "_") < 3

	if isSimple {
		suffixes := []string{"_11.A", "_11.C", "_11.F", "_11.H"}
		targetModel := model
		if targetModel == "" {
			targetModel = otaInput
		}

		for _, suffix := range suffixes {
			currentPrefix := otaInput + suffix
			fullVersion := currentPrefix + ".01_0001_197001010000"
			fmt.Printf("Querying for %s\n\n", currentPrefix)

			result := queryIoTServer(fullVersion, targetModel)
			if result != nil {
				displayIoTResult(result)
				fmt.Println()
			} else {
				fmt.Println("No Result\n")
			}
		}
	} else {
		targetModel := model
		if targetModel == "" {
			targetModel = strings.Split(otaInput, "_")[0]
		}
		fullVersion := otaInput
		if strings.Count(otaInput, "_") < 2 {
			fullVersion = otaInput + ".01_0001_197001010000"
		}

		fmt.Printf("Querying for %s\n\n", otaInput)
		result := queryIoTServer(fullVersion, targetModel)
		if result != nil {
			displayIoTResult(result)
		} else {
			fmt.Println("No Result")
		}
	}
}
