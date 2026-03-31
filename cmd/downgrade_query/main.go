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
	var debug int
	pflag.IntVar(&debug, "debug", 0, "Enable (1) or disable (0) debug output")
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 4 {
		fmt.Println("Usage: downgrade_query <OTA_PREFIX> <PrjNum> <snNum> <DUID> [--debug 0/1]")
		os.Exit(1)
	}

	otaPrefix := strings.ToUpper(args[0])
	prjNum := args[1]
	snNum := args[2]
	duid := args[3]

	if !strings.Contains(otaPrefix, "_11.") {
		fmt.Printf("Error: OTA_Prefix '%s' must contain '_11.'\n", otaPrefix)
		os.Exit(1)
	}
	if len(prjNum) != 5 {
		fmt.Printf("Error: PrjNum '%s' must be exactly 5 digits.\n", prjNum)
		os.Exit(1)
	}
	if len(duid) != 64 {
		fmt.Printf("Error: DUID must be 64 characters.\n")
		os.Exit(1)
	}

	url := "https://downgrade.coloros.com/downgrade/query-v3"
	downgrade.RunQuery(url, otaPrefix, prjNum, snNum, duid, debug == 1)
}
