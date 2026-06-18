package ttclient

import "net/http"

// setCommonHeaders applies the always-required UA and the optional Api-Version.
func setCommonHeaders(req *http.Request, userAgent, apiVersion string) {
	if userAgent == "" {
		userAgent = "tastytrade-mcp/0.1"
	}
	req.Header.Set("User-Agent", userAgent)
	if apiVersion != "" {
		req.Header.Set("Api-Version", apiVersion)
	}
}
