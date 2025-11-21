package store

import (
	"fmt"
	"strings"
)

/*
Generic Azure Service Init Helpers
*/

// AzServiceHelper returns a service URI and the stripped token
type AzServiceHelper struct {
	ServiceUri string
	Token      string
}

// AzServiceFromToken for azure the first part of the token __must__ always be the
// identifier of the service e.g. the account name for tableStore or the Vault name for KVSecret or
// AppConfig instance
// take parameter specifies the number of elements to take from the start only
//
// e.g. a value of 2 for take  will take first 2 elements from the slices
//
// For AppConfig or KeyVault we ONLY need the AppConfig instance or KeyVault instance name
func AzServiceFromToken(token string, formatUri string, take int) AzServiceHelper {
	// ensure preceding slash is trimmed
	stringToken := strings.Split(strings.TrimPrefix(token, "/"), "/")
	splitToken := []any{}
	// recast []string slice to an []any
	for _, st := range stringToken {
		splitToken = append(splitToken, st)
	}

	uri := fmt.Sprintf(formatUri, splitToken[0:take]...)
	return AzServiceHelper{ServiceUri: uri, Token: strings.Join(stringToken[take:], "/")}
}
