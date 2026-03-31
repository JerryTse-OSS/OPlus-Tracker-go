// Designed by Jerry Tse
package main

import (
	"fmt"
	"os"
	"strings"

	"oplus-tracker/pkg/downgrade"

	"github.com/spf13/pflag"
)

func main() {
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: downgrade_query-v2 <OTA_PREFIX> <PrjNum>")
		os.Exit(1)
	}

	otaPrefix := strings.ToUpper(args[0])
	prjNum := args[1]

	if !strings.Contains(otaPrefix, "_11.") {
		otaPrefix = otaPrefix + "_11.A"
	}

	if len(prjNum) != 5 {
		fmt.Printf("Error: PrjNum '%s' must be exactly 5 digits.\n", prjNum)
		os.Exit(1)
	}

	duid := strings.Repeat("0", 64)
	url := "https://downgrade.coloros.com/downgrade/query-v2"
	downgrade.RunQuery(url, otaPrefix, prjNum, "", duid, false)
}
