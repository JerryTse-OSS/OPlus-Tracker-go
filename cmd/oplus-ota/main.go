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
	"oplus-tracker/pkg/util"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/pflag"
)

type ComponentInfo struct {
	Name         string
	Version      string
	Link         string
	OriginalLink string
	Size         string
	MD5          string
	AutoURL      string
	ExpiresTime  *time.Time
}

type OpexInfo struct {
	Index       int
	VersionName string
	BusinessCode string
	ZipHash      string
	AutoURL      string
}

type QueryResult struct {
	Success      bool
	ResponseCode int
	Data         map[string]string
	Error        string
	Components   []ComponentInfo
	OpexList     []OpexInfo
	PublishedTime string
}

type QueryConfig struct {
	OTAPrefix      string
	OTAVersion     string
	Model          string
	Region         string
	Gray           int
	Mode           string
	GUID           string
	ComponentsInput string
	Anti           int
	HasCustomModel bool
	Genshin        string
	Pre            string
	CustomLanguage string
	NVID           string
}

func main() {
	var cfg QueryConfig
	var graynew int

	pflag.StringVar(&cfg.Model, "model", "unknown", "Custom model")
	pflag.StringVar(&cfg.Mode, "mode", "manual", "Query mode (manual, client_auto, server_auto, taste)")
	pflag.StringVar(&cfg.CustomLanguage, "cl", "", "Custom language (e.g. zh-CN)")
	pflag.IntVar(&cfg.Gray, "gray", 0, "Gray update")
	pflag.StringVar(&cfg.Genshin, "genshin", "0", "Genshin edition (0, 1, 2)")
	pflag.StringVar(&cfg.Pre, "pre", "0", "Preview edition (0, 1)")
	pflag.StringVar(&cfg.GUID, "guid", strings.Repeat("0", 64), "64-char device GUID")
	pflag.StringVar(&cfg.ComponentsInput, "components", "", "Custom components (name:version,...)")
	pflag.IntVar(&cfg.Anti, "anti", 0, "Anti mode (0, 1)")
	pflag.StringVar(&cfg.NVID, "nvid", "", "Custom NV Carrier ID (8 digits)")
	pflag.IntVar(&graynew, "graynew", 0, "Query FWs not in taste mode but in gray server")

	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: oplus-ota <OTA_PREFIX> <REGION> [options]")
		fmt.Println("Example: oplus-ota PJX110_11.A cn --anti 1")
		os.Exit(1)
	}

	cfg.OTAPrefix = args[0]
	cfg.Region = strings.ToLower(args[1])
	cfg.HasCustomModel = pflag.Lookup("model").Changed

	if cfg.Pre == "1" && cfg.GUID == strings.Repeat("0", 64) {
		fmt.Println("Error: GUID required for pre mode")
		os.Exit(1)
	}

	if cfg.NVID != "" && (len(cfg.NVID) != 8 || !isDigit(cfg.NVID)) {
		fmt.Println("Error: --nvid must be exactly 8 digits")
		os.Exit(1)
	}

	if graynew == 1 {
		runGrayNew(cfg)
		return
	}

	otaUpper := strings.ToUpper(cfg.OTAPrefix)
	// Special handle for Ovt
	otaUpper = strings.ReplaceAll(otaUpper, "OVT", "Ovt")

	processedOTA, processedModel := processOTAVersion(otaUpper, cfg.Region, cfg.Genshin, cfg.Pre, cfg.Model, cfg.HasCustomModel)
	cfg.OTAVersion = processedOTA
	cfg.Model = processedModel

	isSimpleVersion := regexp.MustCompile(`_\d{2}\.[A-Z]`).MatchString(otaUpper) || strings.Count(otaUpper, "_") >= 3

	if !isSimpleVersion {
		autoCompleteQuery(otaUpper, cfg)
	} else {
		fmt.Printf("\nQuerying %s update\n", strings.ToUpper(cfg.Region))
		fmt.Printf("Device Model: %s\n", cfg.Model)
		fmt.Printf("Full OTA Version: %s\n", cfg.OTAVersion)
		if cfg.GUID == strings.Repeat("0", 64) {
			fmt.Println("Using GUID: Default device ID")
		} else {
			fmt.Printf("Using GUID: %s\n", cfg.GUID[:16])
		}

		result := queryUpdate(cfg)
		if !result.Success && result.ResponseCode == 2004 && cfg.Region == "in" && !cfg.HasCustomModel {
			cfg.Model += "IN"
			result = queryUpdate(cfg)
		}
		displayResult(result)
	}
}

func isDigit(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func runGrayNew(cfg QueryConfig) {
	tasteConfig := cfg
	tasteConfig.Mode = "taste"
	processedOTA, processedModel := processOTAVersion(tasteConfig.OTAPrefix, tasteConfig.Region, tasteConfig.Genshin, tasteConfig.Pre, tasteConfig.Model, tasteConfig.HasCustomModel)
	tasteConfig.OTAVersion = processedOTA
	tasteConfig.Model = processedModel

	resultTaste := queryUpdate(tasteConfig)
	if !resultTaste.Success || resultTaste.Data == nil {
		fmt.Println("Failed to get OTA version")
		os.Exit(1)
	}

	newOTAVersion := resultTaste.Data["ota_version"]
	if newOTAVersion == "" || newOTAVersion == "N/A" {
		fmt.Println("No OTA version found in response")
		os.Exit(1)
	}

	finalConfig := cfg
	finalConfig.OTAVersion = newOTAVersion
	finalConfig.Gray = 1
	finalConfig.Genshin = "0"
	finalConfig.Pre = "0"

	processedOTA2, processedModel2 := processOTAVersion(finalConfig.OTAVersion, finalConfig.Region, "0", "0", finalConfig.Model, finalConfig.HasCustomModel)
	finalConfig.OTAVersion = processedOTA2
	finalConfig.Model = processedModel2

	fmt.Printf("Querying %s update\n", strings.ToUpper(finalConfig.Region))
	fmt.Printf("Device Model: %s\n", finalConfig.Model)
	fmt.Printf("Full OTA Version: %s\n", finalConfig.OTAVersion)
	displayResult(queryUpdate(finalConfig))
}

func processOTAVersion(otaPrefix, region, genshin, pre, customModel string, hasCustomModel bool) (string, string) {
	parts := strings.Split(otaPrefix, "_")
	baseModel := parts[0]
	var model string

	if hasCustomModel {
		model = customModel
	} else if region == "eu" || region == "ru" || region == "tr" {
		model = baseModel + strings.ToUpper(region)
	} else {
		model = baseModel
	}

	otaPrefixMod := otaPrefix
	if genshin == "1" && !strings.Contains(otaPrefix, "YS") {
		model = baseModel
		otaPrefixMod = strings.Replace(otaPrefix, model, model+"YS", 1)
	} else if genshin == "2" && !strings.Contains(otaPrefix, "Ovt") {
		model = baseModel
		otaPrefixMod = strings.Replace(otaPrefix, model, model+"Ovt", 1)
	} else if pre == "1" && !strings.Contains(otaPrefix, "PRE") {
		model = baseModel
		otaPrefixMod = strings.Replace(otaPrefix, model, model+"PRE", 1)
	}

	if strings.Contains(otaPrefixMod, "YS") {
		model = strings.ReplaceAll(baseModel, "YS", "")
	} else if strings.Contains(otaPrefixMod, "Ovt") {
		model = strings.ReplaceAll(baseModel, "Ovt", "")
	} else if strings.Contains(otaPrefixMod, "PRE") {
		model = strings.ReplaceAll(baseModel, "PRE", "")
	}

	otaVersion := otaPrefixMod
	if len(parts) < 3 {
		otaVersion = otaPrefixMod + ".01_0001_197001010000"
	}

	return otaVersion, model
}

func autoCompleteQuery(baseOTAPrefix string, cfg QueryConfig) {
	suffixes := []string{"_11.A", "_11.C", "_11.F", "_11.H", "_11.J"}
	var lastSuccessFake string

	if cfg.Anti == 1 {
		cfg.Mode = "taste"
	}

	for _, suffix := range suffixes {
		displayOTA := baseOTAPrefix + suffix
		decor := ""
		if cfg.Genshin == "1" && !strings.Contains(baseOTAPrefix, "YS") {
			decor = "YS"
		} else if cfg.Genshin == "2" && !strings.Contains(baseOTAPrefix, "Ovt") {
			decor = "Ovt"
		} else if cfg.Pre == "1" && !strings.Contains(baseOTAPrefix, "PRE") {
			decor = "PRE"
		}

		displayName := displayOTA
		if decor != "" {
			displayName = strings.Replace(displayOTA, baseOTAPrefix, baseOTAPrefix+decor, 1)
		}
		fmt.Printf("\nQuerying for %s\n", displayName)

		processedOTA, processedModel := processOTAVersion(displayOTA, cfg.Region, cfg.Genshin, cfg.Pre, cfg.Model, cfg.HasCustomModel)
		currentConfig := cfg
		currentConfig.OTAVersion = processedOTA
		currentConfig.Model = processedModel

		result := queryUpdate(currentConfig)

		if !result.Success && result.ResponseCode == 2004 && cfg.Region == "in" && !cfg.HasCustomModel {
			currentConfig.Model = processedModel + "IN"
			result = queryUpdate(currentConfig)
		}

		if cfg.Anti == 1 && !result.Success && result.ResponseCode == 2004 && lastSuccessFake != "" {
			retryOTA, retryModel := processOTAVersion(lastSuccessFake, cfg.Region, cfg.Genshin, cfg.Pre, cfg.Model, cfg.HasCustomModel)
			retryConfig := cfg
			retryConfig.OTAVersion = retryOTA
			retryConfig.Model = retryModel
			retryConfig.Anti = 0

			result = queryUpdate(retryConfig)
			if !result.Success && result.ResponseCode == 2004 && cfg.Region == "in" && !cfg.HasCustomModel {
				retryConfig.Model = retryModel + "IN"
				result = queryUpdate(retryConfig)
			}
		}

		if result.Success && cfg.Anti == 1 {
			fake := result.Data["fake_ota_version"]
			if fake != "" && fake != "N/A" {
				lastSuccessFake = fake
			}
		}

		displayResult(result)
	}
}

func queryUpdate(cfg QueryConfig) QueryResult {
	keyRegion := cfg.Region
	if keyRegion != "cn" && keyRegion != "eu" && keyRegion != "in" {
		keyRegion = "sg"
	}

	pubKeyPEM := config.PUBLIC_KEYS[keyRegion]
	
	regConfig := config.REGION_CONFIG[cfg.Region]
	if cfg.Gray == 1 && cfg.Region == "cn" {
		regConfig = config.REGION_CONFIG["cn_gray"]
	} else if cfg.Region != "cn" && cfg.Region != "eu" && cfg.Region != "in" {
		// Use sg_host as base
		base := config.REGION_CONFIG["sg_host"]
		base.Language = regConfig.Language
		base.CarrierID = regConfig.CarrierID
		regConfig = base
	}

	aesKey := crypto.GenerateRandomBytes(32)
	iv := crypto.GenerateRandomBytes(16)
	deviceID := crypto.GenerateRandomString(64)

	// In Python: base64.b64encode(aes_key) then encrypt
	aesKeyB64 := base64.StdEncoding.EncodeToString(aesKey)
	protectedKey, err := crypto.RSAEncryptOAEP([]byte(aesKeyB64), pubKeyPEM)
	if err != nil {
		return QueryResult{Error: "Encryption failed: " + err.Error()}
	}

	headers := map[string]string{
		"language":       cfg.CustomLanguage,
		"newLanguage":    cfg.CustomLanguage,
		"androidVersion": "unknown",
		"colorOSVersion": "unknown",
		"romVersion":     "unknown",
		"infVersion":     "1",
		"otaVersion":     cfg.OTAVersion,
		"model":          cfg.Model,
		"mode":           cfg.Mode,
		"nvCarrier":      regConfig.CarrierID,
		"pipelineKey":    "ALLNET",
		"operator":       "ALLNET",
		"companyId":      "",
		"version":        "2",
		"deviceId":       deviceID,
		"Content-Type":   "application/json; charset=utf-8",
	}
	if cfg.CustomLanguage == "" {
		headers["language"] = regConfig.Language
		headers["newLanguage"] = regConfig.Language
	}
	if cfg.NVID != "" {
		headers["nvCarrier"] = cfg.NVID
	}

	expiry := time.Now().Add(24 * time.Hour).UnixNano() / int64(time.Millisecond) // approximate
	protectedKeyMap := map[string]interface{}{
		"SCENE_1": map[string]interface{}{
			"protectedKey":       protectedKey,
			"version":            fmt.Sprintf("%d", expiry),
			"negotiationVersion": regConfig.PublicKeyVersion,
		},
	}
	pkJSON, _ := json.Marshal(protectedKeyMap)
	headers["protectedKey"] = string(pkJSON)

	requestBody := map[string]interface{}{
		"mode":     "0",
		"time":     time.Now().UnixNano() / int64(time.Millisecond),
		"isRooted": "0",
		"isLocked": true,
		"type":     "0",
		"deviceId": strings.ToLower(cfg.GUID),
		"opex":     map[string]bool{"check": true},
	}

	if cfg.ComponentsInput != "" {
		requestBody["components"] = parseComponents(cfg.ComponentsInput)
	}

	rbJSON, _ := json.Marshal(requestBody)
	cipherText, _ := crypto.AESCTREncrypt(rbJSON, aesKey, iv)

	params := map[string]string{
		"cipher": base64.StdEncoding.EncodeToString(cipherText),
		"iv":     base64.StdEncoding.EncodeToString(iv),
	}
	paramsJSON, _ := json.Marshal(params)

	client := resty.New()
	endpointVer := "/update/v3"
	if cfg.Pre == "1" || (cfg.GUID != "" && cfg.GUID != strings.Repeat("0", 64)) {
		endpointVer = "/update/v6"
	}
	url := fmt.Sprintf("https://%s%s", regConfig.Host, endpointVer)

	var resp *resty.Response
	for i := 0; i < 3; i++ {
		resp, err = client.R().
			SetHeaders(headers).
			SetBody(map[string]string{"params": string(paramsJSON)}).
			Post(url)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(5*(i+1)) * time.Second)
	}

	if err != nil {
		return QueryResult{Error: "Request failed: " + err.Error()}
	}

	return processResponse(resp, aesKey)
}

func parseComponents(input string) []map[string]string {
	var res []map[string]string
	for _, pair := range strings.Split(input, ",") {
		parts := strings.Split(pair, ":")
		if len(parts) == 2 {
			res = append(res, map[string]string{
				"componentName":    strings.TrimSpace(parts[0]),
				"componentVersion": strings.TrimSpace(parts[1]),
			})
		}
	}
	return res
}

func processResponse(resp *resty.Response, aesKey []byte) QueryResult {
	if resp == nil {
		return QueryResult{Error: "No response"}
	}

	if !resp.IsSuccess() || !strings.Contains(resp.Header().Get("Content-Type"), "application/json") {
		return QueryResult{Error: fmt.Sprintf("Server returned non-JSON response\nStatus Code: %d\nResponse: %s", resp.StatusCode(), resp.String())}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return QueryResult{Error: "Invalid JSON response"}
	}

	code, _ := result["responseCode"].(float64)
	if code != 200 {
		errMsg, _ := result["error"].(string)
		return QueryResult{Success: false, ResponseCode: int(code), Error: errMsg}
	}

	bodyStr, _ := result["body"].(string)
	var encryptedBody map[string]string
	json.Unmarshal([]byte(bodyStr), &encryptedBody)

	cipherBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["cipher"])
	ivBytes, _ := base64.StdEncoding.DecodeString(encryptedBody["iv"])

	decrypted, _ := crypto.AESCTRDecrypt(cipherBytes, aesKey, ivBytes)
	var body map[string]interface{}
	json.Unmarshal(decrypted, &body)

	res := QueryResult{Success: true, ResponseCode: 200, Data: make(map[string]string)}

	if ts, ok := body["publishedTime"].(float64); ok {
		res.PublishedTime = time.Unix(int64(ts)/1000, 0).Format("2006-01-02 15:04:05")
	}

	compList, _ := body["components"].([]interface{})
	for _, c := range compList {
		comp, _ := c.(map[string]interface{})
		pkts, _ := comp["componentPackets"].(map[string]interface{})

		manualURL, _ := pkts["manualUrl"].(string)
		autoURL, _ := pkts["url"].(string)
		manualURL = util.ReplaceGaussURL(manualURL)
		autoURL = util.ReplaceGaussURL(autoURL)

		finalLink := manualURL
		var expires *time.Time

		if strings.Contains(manualURL, "downloadCheck") {
			finalLink = util.ReplaceGaussURL(util.GetRedirectURL(manualURL, 3))
			expires = extractExpirationDate(finalLink)
		}

		size, _ := pkts["size"].(string)
		md5, _ := pkts["md5"].(string)

		res.Components = append(res.Components, ComponentInfo{
			Name:         comp["componentName"].(string),
			Version:      comp["componentVersion"].(string),
			Link:         finalLink,
			OriginalLink: manualURL,
			Size:         size,
			MD5:          md5,
			AutoURL:      autoURL,
			ExpiresTime:  expires,
		})
	}

	opexInfo, _ := body["opex"].(map[string]interface{})
	if opexInfo != nil {
		opexPackages, _ := opexInfo["opexPackage"].([]interface{})
		for i, p := range opexPackages {
			pkg, _ := p.(map[string]interface{})
			if pkg["code"].(float64) == 200 {
				info, _ := pkg["info"].(map[string]interface{})
				res.OpexList = append(res.OpexList, OpexInfo{
					Index:       i + 1,
					VersionName: opexInfo["opexVersionName"].(string),
					BusinessCode: pkg["businessCode"].(string),
					ZipHash:      info["zipHash"].(string),
					AutoURL:      util.ReplaceGaussURL(info["autoUrl"].(string)),
				})
			}
		}
	}

	desc, _ := body["description"].(map[string]interface{})
	changelog := "N/A"
	if desc != nil {
		changelog, _ = desc["panelUrl"].(string)
	}
	res.Data["changelog"] = util.ReplaceGaussURL(changelog)
	res.Data["security_patch"], _ = body["securityPatch"].(string)
	
	realVersion, _ := body["realVersionName"].(string)
	if realVersion == "" {
		realVersion, _ = body["versionName"].(string)
	}
	res.Data["version"] = realVersion
	
	otaVer, _ := body["otaVersion"].(string)
	res.Data["fake_ota_version"] = otaVer
	
	realOTAVer, _ := body["realOtaVersion"].(string)
	if realOTAVer == "" {
		realOTAVer = otaVer
	}
	res.Data["ota_version"] = realOTAVer

	return res
}

func extractExpirationDate(url string) *time.Time {
	re := regexp.MustCompile(`(Expires|x-oss-expires)=(\d+)`)
	match := re.FindStringSubmatch(url)
	if len(match) == 3 {
		ts := match[2]
		var timestamp int64
		fmt.Sscanf(ts, "%d", &timestamp)
		t := time.Unix(timestamp, 0)
		return &t
	}
	return nil
}

func displayResult(result QueryResult) {
	if result.Success {
		fmt.Println("\nFetch Info:")
		if len(result.Components) > 0 {
			if len(result.Components) == 1 {
				fmt.Printf("• Link: %s\n", result.Components[0].Link)
			} else {
				for i, comp := range result.Components {
					fmt.Printf("\nComponent %d: %s\n", i+1, comp.Name)
					fmt.Printf("Link: %s\n", comp.Link)
					fmt.Printf("MD5: %s\n", comp.MD5)
				}
			}
		}

		fmt.Printf("• Changelog: %s\n", result.Data["changelog"])
		if result.PublishedTime != "" {
			fmt.Printf("• Published Time: %s\n", result.PublishedTime)
		}
		fmt.Printf("• Security Patch: %s\n", result.Data["security_patch"])
		fmt.Printf("• Version: %s\n", result.Data["version"])
		fmt.Printf("• Ota Version: %s\n", result.Data["ota_version"])

		if len(result.Components) > 0 && result.Components[0].ExpiresTime != nil {
			fmt.Printf("\n• Notice: Dynamic Link will expire at %s\n", result.Components[0].ExpiresTime.Format("2006-01-02 15:04:05"))
		}

		if len(result.OpexList) > 0 {
			fmt.Println("\nOpex Info:")
			for i, opex := range result.OpexList {
				fmt.Printf("• Link: %s\n", opex.AutoURL)
				fmt.Printf("• Zip Hash: %s\n", opex.ZipHash)
				fmt.Printf("• Opex Codename: %s\n", opex.BusinessCode)
				fmt.Printf("• Opex Version Name: %s\n", opex.VersionName)
				if i < len(result.OpexList)-1 {
					fmt.Println()
				}
			}
		}
	} else {
		if result.ResponseCode == 2004 {
			fmt.Println("\nNo Result")
		} else if result.ResponseCode == 308 {
			fmt.Println("\nFlow Limit\nTry again later")
		} else if result.ResponseCode == 500 {
			fmt.Printf("\nServer Error (Code 500)\nError: %s\n", result.Error)
		} else if result.ResponseCode == 204 || result.ResponseCode == 2200 {
			fmt.Println("\nCurrent IMEI is not in test IMEI set")
		} else if result.Error != "" {
			fmt.Printf("\nError: %s\n", result.Error)
		} else {
			fmt.Println("\nUnknown Error")
		}
	}
}
