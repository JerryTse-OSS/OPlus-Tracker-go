// Designed by Jerry Tse
package sota

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"oplus-tracker/pkg/config"
	"oplus-tracker/pkg/crypto"

	"github.com/go-resty/resty/v2"
)

const (
	API_URL_QUERY       = "https://component-ota-cn.allawntech.com/update/v6"
	API_URL_UPDATE      = "https://component-ota-cn.allawntech.com/sotaUpdate/v1"
	API_URL_DESCRIPTION = "https://component-ota-cn.allawntech.com/sotaDescriptionInfo/v2"
)

type SotaConfig struct {
	Brand      string
	OTAVersion string
	Model      string
	ColorOS    string
	RomVersion string
}

func BuildHeaders(aesKey []byte, cfg SotaConfig, isUpdate bool) map[string]string {
	aesKeyB64 := base64.StdEncoding.EncodeToString(aesKey)
	protectedKey, _ := crypto.RSAEncryptOAEP([]byte(aesKeyB64), config.PUBLIC_KEYS["cn"])

	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()+1000000000*60*60*24)
	pkMap := map[string]interface{}{
		"SCENE_1": map[string]interface{}{
			"protectedKey":       protectedKey,
			"version":            timestamp,
			"negotiationVersion": "1615879139745",
		},
	}
	pkJSON, _ := json.Marshal(pkMap)

	brandLower := strings.ToLower(cfg.Brand)
	brandOrig := cfg.Brand
	if brandLower == "oneplus" {
		brandOrig = "OnePlus"
	} else if brandLower == "oppo" {
		brandOrig = "OPPO"
	} else if brandLower == "realme" {
		brandOrig = "realme"
	}

	headers := map[string]string{
		"language":       "zh-CN",
		"colorOSVersion": cfg.ColorOS,
		"androidVersion": "unknown",
		"infVersion":     "1",
		"otaVersion":     cfg.OTAVersion,
		"model":          cfg.Model,
		"mode":           "taste",
		"nvCarrier":      "10010111",
		"brand":          brandOrig,
		"brandSota":      brandOrig,
		"osType":         "domestic_" + brandOrig,
		"version":        "2",
		"deviceId":       fmt.Sprintf("%064d", 0),
		"protectedKey":   string(pkJSON),
		"Content-Type":   "application/json; charset=utf-8",
		"User-Agent":     "okhttp/4.12.0",
	}

	if isUpdate {
		headers["romVersion"] = cfg.RomVersion
	} else {
		headers["romVersion"] = "unknown"
	}

	return headers
}

func Query(cfg SotaConfig) (map[string]interface{}, []byte, error) {
	aesKey := crypto.GenerateRandomBytes(32)
	iv := crypto.GenerateRandomBytes(16)

	headers := BuildHeaders(aesKey, cfg, false)

	now := time.Now().UnixNano() / 1e6
	body := map[string]interface{}{
		"mode":     0,
		"time":     now,
		"isRooted": "0",
		"isLocked": true,
		"type":     "1",
		"securityPatch": "1970-01-01",
		"securityPatchVendor": "1970-01-01",
		"cota": map[string]string{
			"cotaVersion": "", "cotaVersionName": "", "buildType": "user",
		},
		"opex": map[string]bool{"check": true},
		"sota": map[string]interface{}{
			"sotaProtocolVersion": "2",
			"sotaVersion":         "V69P69",
			"otaUpdateTime":       now - (15 * 24 * 60 * 60 * 1000),
			"frameworkVer":        "10",
			"supportLightH":       "1",
			"updateViaReboot":     2,
			"sotaProtocolVersionNew": []string{"apk", "opex", "rus"},
		},
		"otaAppVersion": 16000021,
		"deviceId":      fmt.Sprintf("%064d", 0),
	}

	rbJSON, _ := json.Marshal(body)
	cipherText, _ := crypto.AESCTREncrypt(rbJSON, aesKey, iv)

	params := map[string]string{
		"cipher": base64.StdEncoding.EncodeToString(cipherText),
		"iv":     base64.StdEncoding.EncodeToString(iv),
	}
	paramsJSON, _ := json.Marshal(params)

	client := resty.New()
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(map[string]string{"params": string(paramsJSON)}).
		Post(API_URL_QUERY)

	if err != nil {
		return nil, nil, err
	}

	var res map[string]interface{}
	json.Unmarshal(resp.Body(), &res)

	bodyStr, _ := res["body"].(string)
	var encryptedBody map[string]string
	json.Unmarshal([]byte(bodyStr), &encryptedBody)

	cipherBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["cipher"])
	ivBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["iv"])

	if len(ivBytes) != 16 {
		return nil, nil, fmt.Errorf("server returned invalid IV length: %d", len(ivBytes))
	}

	decrypted, _ := crypto.AESCTRDecrypt(cipherBytes, aesKey, ivBytes)

	var decryptedJSON map[string]interface{}
	json.Unmarshal(decrypted, &decryptedJSON)

	return decryptedJSON, aesKey, nil
}

func Update(queryResult map[string]interface{}, cfg SotaConfig) (map[string]interface{}, error) {
	sotaData, _ := queryResult["sota"].(map[string]interface{})
	if sotaData == nil {
		return nil, fmt.Errorf("no SOTA data")
	}

	newSotaVersion, _ := sotaData["sotaVersion"].(string)
	moduleMap, _ := sotaData["moduleMap"].(map[string]interface{})
	apkModules, _ := moduleMap["apk"].([]interface{})

	var sauModules []map[string]interface{}
	for _, m := range apkModules {
		mod, _ := m.(map[string]interface{})
		name := mod["moduleName"].(string)
		latestVer, _ := mod["moduleVersion"].(float64)

		currentVer := latestVer - 1
		if latestVer > 100 {
			currentVer = latestVer - (latestVer / 10)
		}
		if currentVer < 1 {
			currentVer = 1
		}

		sauModules = append(sauModules, map[string]interface{}{
			"sotaVersion":   newSotaVersion,
			"moduleName":    name,
			"moduleVersion": int(currentVer),
		})
	}

	body := map[string]interface{}{
		"sotaProtocolVersion":    "2",
		"sotaProtocolVersionNew": []string{"apk", "opex", "rus"},
		"sotaVersion":            "V69P69",
		"updateViaReboot":        2,
		"supportLightH":          "1",
		"moduleMap": map[string]interface{}{
			"sau": sauModules,
		},
		"mode":       0,
		"deviceId":   fmt.Sprintf("%064d", 0),
		"otaVersion": cfg.OTAVersion,
	}

	aesKey := crypto.GenerateRandomBytes(32)
	iv := crypto.GenerateRandomBytes(16)
	headers := BuildHeaders(aesKey, cfg, true)

	rbJSON, _ := json.Marshal(body)
	cipherText, _ := crypto.AESCTREncrypt(rbJSON, aesKey, iv)

	params := map[string]string{
		"cipher": base64.StdEncoding.EncodeToString(cipherText),
		"iv":     base64.StdEncoding.EncodeToString(iv),
	}
	paramsJSON, _ := json.Marshal(params)

	client := resty.New()
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(map[string]string{"params": string(paramsJSON)}).
		Post(API_URL_UPDATE)

	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	json.Unmarshal(resp.Body(), &res)

	bodyStr, _ := res["body"].(string)
	var encryptedBody map[string]string
	json.Unmarshal([]byte(bodyStr), &encryptedBody)

	cipherBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["cipher"])
	ivBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["iv"])

	if len(ivBytes) != 16 {
		return nil, fmt.Errorf("server returned invalid IV length: %d", len(ivBytes))
	}

	decrypted, _ := crypto.AESCTRDecrypt(cipherBytes, aesKey, ivBytes)

	var decryptedJSON map[string]interface{}
	json.Unmarshal(decrypted, &decryptedJSON)

	return decryptedJSON, nil
}

func FetchDescription(modules []map[string]interface{}, sotaVersion string, cfg SotaConfig) (map[string]interface{}, error) {
	var sotaList []map[string]interface{}
	for _, mod := range modules {
		sotaList = append(sotaList, map[string]interface{}{
			"sotaVersion":   sotaVersion,
			"moduleName":    mod["moduleName"],
			"moduleVersion": mod["moduleVersion"],
		})
	}

	innerParams := map[string]interface{}{
		"otaVersion":          cfg.OTAVersion,
		"mode":                0,
		"deviceId":            fmt.Sprintf("%064d", 0),
		"sota":                sotaList,
		"sotaProtocolVersion": "2",
		"sotaVersion":         sotaVersion,
		"noUpgradeModules":    []interface{}{},
		"h5LinkVersion":       6,
	}
	paramsStr, _ := json.Marshal(innerParams)

	brandLower := strings.ToLower(cfg.Brand)
	headers := map[string]string{
		"language":           "zh-CN",
		"brandSota":          brandLower,
		"sec-ch-ua-platform": "Android",
		"colorOSVersion":     cfg.ColorOS,
		"osType":             "domestic_" + cfg.Brand,
		"romVersion":         sotaVersion,
		"nvCarrier":          "10010111",
		"mode":               "manual",
		"osVersion":          cfg.ColorOS,
		"otaVersion":         cfg.OTAVersion,
		"model":              cfg.Model,
		"uRegion":            "undefined",
		"androidVersion":     "unknown",
		"Accept":             "application/json, text/plain, */*",
		"Content-Type":       "application/json",
		"User-Agent":         "okhttp/4.12.0",
	}

	client := resty.New()
	resp, err := client.R().
		SetHeaders(headers).
		SetBody(map[string]string{"params": string(paramsStr)}).
		Post(API_URL_DESCRIPTION)

	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	json.Unmarshal(resp.Body(), &res)
	return res, nil
}
