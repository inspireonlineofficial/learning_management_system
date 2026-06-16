package main

import (
	"encoding/pem"
	"fmt"
	"strings"
)

func main() {
	// Let's use the exact string from docker inspect (which has double quotes at the end, but wait, does it?)
	// In docker inspect, the string had literal newlines.
	// Let's simulate:
	rawPub := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAtj5BWE4kfHw8rc0FoXYn
t4gDGv0ugOav3q2VBDGOXUsCDffnD/bjcIPtWC9eMReWv7qPyGfqMV7ZrB8ki85O
yshAS5/aQkHCal9cqVFHPNogpW9tCvL9afOqZlZo7WU3vNNEAyl2CjEXms0jRtlb
nTaaAf+J3NoM8xiBxV5nIsaQgXDXO9q9hZQv353RfFLrJ1T4kNJ3i5ciESNMxatU
geFNdd6+3ucw+vGmKLF9WKM5tIPlVpxPaKTOZZtSbTrXo3t2Dol+NTxvGKTPwJe3
lzpAh6eDO24zgTfAupkm0bZhdEmvUF36FuIeP6MBOoXFDPhKKDuanzwh5yrsawUa
DQIDAQAB
-----END PUBLIC KEY-----`

	// What if it is wrapped in double quotes?
	rawPubWithQuotes := `"` + rawPub + `"`

	// Let's test pem.Decode on rawPub
	block1, _ := pem.Decode([]byte(rawPub))
	if block1 == nil {
		fmt.Println("Raw public key decode failed")
	} else {
		fmt.Println("Raw public key decode succeeded")
	}

	// Let's test pem.Decode on rawPubWithQuotes
	block2, _ := pem.Decode([]byte(rawPubWithQuotes))
	if block2 == nil {
		fmt.Println("Raw public key with quotes decode failed")
	} else {
		fmt.Println("Raw public key with quotes decode succeeded")
	}

	// What if we trim quotes?
	trimmed := strings.Trim(rawPubWithQuotes, "\"")
	block3, _ := pem.Decode([]byte(trimmed))
	if block3 == nil {
		fmt.Println("Trimmed public key decode failed")
	} else {
		fmt.Println("Trimmed public key decode succeeded")
	}
}
