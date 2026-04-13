// Designed by Jerry Tse
package config

type RegionConfig struct {
	Host             string
	Language         string
	CarrierID        string
	PublicKeyVersion string
}

var REGION_CONFIG = map[string]RegionConfig{
	"cn":      {Host: "component-ota-cn.allawntech.com", Language: "zh-CN", CarrierID: "10010111", PublicKeyVersion: "1615879139745"},
	"cn_cmcc": {Host: "component-ota-cn.allawntech.com", Language: "zh-CN", CarrierID: "10011000", PublicKeyVersion: "1615879139745"},
	"cn_gray": {Host: "component-ota-gray.coloros.com", Language: "zh-CN", CarrierID: "10010111", PublicKeyVersion: "1615879139745"},
	"eu":      {Host: "component-ota-eu.allawnos.com", Language: "en-GB", CarrierID: "01000100", PublicKeyVersion: "1615897067573"},
	"in":      {Host: "component-ota-in.allawnos.com", Language: "en-IN", CarrierID: "00011011", PublicKeyVersion: "1615896309308"},
	"sg_host": {Host: "component-ota-sg.allawnos.com", PublicKeyVersion: "1615895993238"},
	"sg":      {Language: "en-SG", CarrierID: "01011010"},
	"ru":      {Language: "ru-RU", CarrierID: "00110111"},
	"tr":      {Language: "tr-TR", CarrierID: "01010001"},
	"th":      {Language: "th-TH", CarrierID: "00111001"},
	"gl":      {Language: "en-US", CarrierID: "10100111"},
	"id":      {Language: "id-ID", CarrierID: "00110011"},
	"tw":      {Language: "zh-TW", CarrierID: "00011010"},
	"my":      {Language: "ms-MY", CarrierID: "00111000"},
	"vn":      {Language: "vi-VN", CarrierID: "00111100"},
	"sa":      {Language: "sa-SA", CarrierID: "10000011"},
	"mea":     {Language: "en-MEA", CarrierID: "10100110"},
	"ph":      {Language: "en-PH", CarrierID: "001111110"},
	"roe":     {Language: "en-EU", CarrierID: "10001101"},
	"la":      {Language: "en-LA", CarrierID: "10011010"},
	"br":      {Language: "en-BR", CarrierID: "10011110"},
}

var IOT_OLD_KEYS = []string{"oppo1997", "baed2017", "java7865", "231uiedn", "09e32ji6",
	"0oiu3jdy", "0pej387l", "2dkliuyt", "20odiuye", "87j3id7w"}

const (
	IOT_SPECIAL_SERVER_CN = "https://iota.coloros.com/post/Query_Update"
	GAUSS_AUTO_URL        = "https://gauss-otacostauto-cn.allawnfs.com/"
	GAUSS_MANUAL_URL      = "https://gauss-componentotacostmanual-cn.allawnfs.com/"
)

var PUBLIC_KEYS = map[string]string{
	"cn": `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApXYGXQpNL7gmMzzvajHa
oZIHQQvBc2cOEhJc7/tsaO4sT0unoQnwQKfNQCuv7qC1Nu32eCLuewe9LSYhDXr9
KSBWjOcCFXVXteLO9WCaAh5hwnUoP/5/Wz0jJwBA+yqs3AaGLA9wJ0+B2lB1vLE4
FZNE7exUfwUc03fJxHG9nCLKjIZlrnAAHjRCd8mpnADwfkCEIPIGhnwq7pdkbamZ
coZfZud1+fPsELviB9u447C6bKnTU4AaMcR9Y2/uI6TJUTcgyCp+ilgU0JxemrSI
PFk3jbCbzamQ6Shkw/jDRzYoXpBRg/2QDkbq+j3ljInu0RHDfOeXf3VBfHSnQ66H
CwIDAQAB
-----END RSA PUBLIC KEY-----`,
	"eu": `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAh8/EThsK3f0WyyPgrtXb
/D0Xni6UZNppaQHUqHWo976cybl92VxmehE0ISObnxERaOtrlYmTPIxkVC9MMueD
vTwZ1l0KxevZVKU0sJRxNR9AFcw6D7k9fPzzpNJmhSlhpNbt3BEepdgibdRZbacF
3NWy3ejOYWHgxC+I/Vj1v7QU5gD+1OhgWeRDcwuV4nGY1ln2lvkRj8EiJYXfkSq/
wUI5AvPdNXdEqwou4FBcf6mD84G8pKDyNTQwwuk9lvFlcq4mRqgYaFg9DAgpDgqV
K4NTJWM7tQS1GZuRA6PhupfDqnQExyBFhzCefHkEhcFywNyxlPe953NWLFWwbGvF
KwIDAQAB
-----END RSA PUBLIC KEY-----`,
	"in": `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwYtghkzeStC9YvAwOQmW
ylbp74Tj8hhi3f9IlK7A/CWrGbLgzz/BeKxNb45zBN8pgaaEOwAJ1qZQV5G4nPro
WCPOP1ro1PkemFJvw/vzOOT5uN0ADnHDzZkZXCU/knxqUSfLcwQlHXsYhNsAm7uO
KjY9YXF4zWzYN0eFPkML3Pj/zg7hl/ov9clB2VeyI1/blMHFfcNA/fvqDTENXcNB
IhgJvXiCpLcZqp+aLZPC5AwY/sCb3j5jTWer0Rk0ZjQBZE1AncwYvUx4mA65U59c
WpTyl4c47J29MsQ66hqWv6eBHlDNZSEsQpHePUqgsf7lmO5Wd7teB8ugQki2oz1Y
5QIDAQAB
-----END RSA PUBLIC KEY-----`,
	"sg": `-----BEGIN RSA PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAkA980wxi+eTGcFDiw2I6
RrUeO4jL/Aj3Yw4dNuW7tYt+O1sRTHgrzxPD9SrOqzz7G0KgoSfdFHe3JVLPN+U1
waK+T0HfLusVJshDaMrMiQFDUiKajb+QKr+bXQhVofH74fjat+oRJ8vjXARSpFk4
/41x5j1Bt/2bHoqtdGPcUizZ4whMwzap+hzVlZgs7BNfepo24PWPRujsN3uopl+8
u4HFpQDlQl7GdqDYDj2zNOHdFQI2UpSf0aIeKCKOpSKF72KDEESpJVQsqO4nxMwE
i2jMujQeCHyTCjBZ+W35RzwT9+0pyZv8FB3c7FYY9FdF/+lvfax5mvFEBd9jO+dp
MQIDAQAB
-----END RSA PUBLIC KEY-----`,
}
