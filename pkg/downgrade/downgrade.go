// Designed by Jerry Tse
package downgrade

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"oplus-tracker/pkg/crypto"

	"github.com/go-resty/resty/v2"
)

const (
	REAL_PUB_KEY        = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAmeQzr0TIbtwZFnDXgatg
6xP9SlNBFho1NTdFQ27SKDF+dBEEfnG9BqRw0na0DUqtpWe2CUtldbU33nnJ0KB6
z7y5f+89o9n8mJxIbh952gpskBxyrhCfpYHV5mt/n9Tkm8OcQWLRFou7/XITuZeZ
ejfUTesQjpfOeCaeKyVSoKQc6WuH7NSYq6B37RMyEn/1+vo8XuHEKD84p29KGpyG
I7ZeL85iOcwBmOD6+e4yideH2RatA1SzEv/9V8BflaFLAWDuPWUjA2WgfOvy5spY
mp/MoMOX4P0d+AkJ9Ms6PUXEUBsbOACmaMFyLCLHmd18+UeGdJR/3I15sXKbJhKe
rwIDAQAB
-----END PUBLIC KEY-----`
	NEGOTIATION_VERSION = 1636449646204
)

type DowngradePkg struct {
	DownloadURL        string      `json:"downloadUrl"`
	VersionIntroduction string     `json:"versionIntroduction"`
	ColorOSVersion     string      `json:"colorosVersion"`
	AndroidVersion     string      `json:"androidVersion"`
	OTAVersion         string      `json:"otaVersion"`
	FileMD5            string      `json:"fileMd5"`
	FileSize           interface{} `json:"fileSize"`
}

type DowngradeResponse struct {
	Code int `json:"code"`
	Data struct {
		DowngradeVoList []DowngradePkg `json:"downgradeVoList"`
		MetaData        string         `json:"metaData"`
	} `json:"data"`
	Cipher string `json:"cipher"`
	IV     string `json:"iv"`
}

func RunQuery(url, otaVersion, prjNum, snNum, duid string, debug bool) {
	model := strings.Split(otaVersion, "_")[0]
	carriers := []string{"10010111", "10011000"}

	fmt.Printf("Querying downgrade for %s\n\n", otaVersion)

	for i, carrier := range carriers {
		sessionKey := make([]byte, 32)
		rand.Read(sessionKey)
		iv := make([]byte, 12)
		rand.Read(iv)

		// Get protected key: base64(aes_key) then RSA encrypt
		sessionKeyB64 := base64.StdEncoding.EncodeToString(sessionKey)
		protectedKey, err := crypto.RSAEncryptOAEP([]byte(sessionKeyB64), REAL_PUB_KEY)
		if err != nil {
			fmt.Printf("[!] Encryption error: %v\n", err)
			return
		}

		encryptedDeviceID, err := crypto.AESGCMEncrypt([]byte(duid), sessionKey, iv, nil)
		if err != nil {
			fmt.Printf("[!] Encryption error: %v\n", err)
			return
		}

		payload := map[string]interface{}{
			"model":      model,
			"nvCarrier":  carrier,
			"prjNum":     prjNum,
			"otaVersion": otaVersion,
			"deviceId": map[string]string{
				"cipher": base64.StdEncoding.EncodeToString(encryptedDeviceID),
				"iv":     base64.StdEncoding.EncodeToString(iv),
			},
		}
		if snNum != "" {
			payload["serialNo"] = snNum
		}

		cipherInfo := map[string]interface{}{
			"downgrade-server": map[string]interface{}{
				"negotiationVersion": int64(NEGOTIATION_VERSION),
				"protectedKey":       protectedKey,
				"version":            fmt.Sprintf("%d", time.Now().Unix()),
			},
		}
		cipherInfoJSON, _ := json.Marshal(cipherInfo)

		client := resty.New()
		resp, err := client.R().
			SetHeaders(map[string]string{
				"Content-Type": "application/json; charset=UTF-8",
				"cipherInfo":   string(cipherInfoJSON),
				"deviceId":     duid,
				"Connection":   "close",
			}).
			SetBody(payload).
			Post(url)

		if err != nil {
			if i == 0 {
				time.Sleep(1 * time.Second)
				continue
			}
			fmt.Printf("[!] Error: %v\n", err)
			break
		}

		var respJSON DowngradeResponse
		if err := json.Unmarshal(resp.Body(), &respJSON); err != nil {
			fmt.Printf("[!] Error parsing response: %v\n", err)
			break
		}

		if respJSON.Code == 1004 {
			fmt.Println("DUID query GUID is empty")
			return
		}

		var finalData DowngradeResponse
		if respJSON.Cipher != "" {
			cipherBytes, _ := base64.StdEncoding.DecodeString(respJSON.Cipher)
			ivBytes, _ := base64.StdEncoding.DecodeString(respJSON.IV)
			decrypted, err := crypto.AESGCMDecrypt(cipherBytes, sessionKey, ivBytes, nil)
			if err == nil {
				json.Unmarshal(decrypted, &finalData)
			}
		} else {
			finalData = respJSON
		}

		if len(finalData.Data.DowngradeVoList) > 0 {
			for j, pkg := range finalData.Data.DowngradeVoList {
				fmt.Println("Fetch Info:")
				fmt.Printf("• Link: %s\n", pkg.DownloadURL)
				fmt.Printf("• Changelog: %s\n", pkg.VersionIntroduction)
				fmt.Printf("• Version: %s (%s)\n", pkg.ColorOSVersion, pkg.AndroidVersion)
				fmt.Printf("• Ota Version: %s\n", pkg.OTAVersion)
				fmt.Printf("• MD5: %s\n", pkg.FileMD5)
				
				fmt.Printf("• File Size: %v Byte", pkg.FileSize)
				if pkg.FileSize != nil {
					// Handle various types for FileSize
					var sizeVal float64
					switch v := pkg.FileSize.(type) {
					case float64: sizeVal = v
					case string: fmt.Sscanf(v, "%f", &sizeVal)
					}
					if sizeVal > 0 {
						fmt.Printf(" (%.0fM)", sizeVal/1024/1024)
					}
				}
				fmt.Println()

				if j < len(finalData.Data.DowngradeVoList)-1 || debug {
					fmt.Println()
				}
			}
			if debug && finalData.Data.MetaData != "" {
				fmt.Printf("Metadata:\n%s\n", finalData.Data.MetaData)
			}
			return
		}

		if i == 0 {
			time.Sleep(1 * time.Second)
			continue
		} else {
			fmt.Println("No Downgrade Package")
		}
	}
}
