package mcptools

import "strings"

// trimSlash removes a trailing slash so we don't emit "//".
func trimSlash(u string) string { return strings.TrimRight(u, "/") }

func protectedResourceDoc(serverURL string) map[string]any {
	s := trimSlash(serverURL)
	return map[string]any{
		"resource":              s + "/mcp",
		"authorization_servers": []string{s},
	}
}

func authServerMetadataDoc(serverURL string) map[string]any {
	s := trimSlash(serverURL)
	return map[string]any{
		"issuer":                                s,
		"authorization_endpoint":                s + "/oauth/authorize",
		"token_endpoint":                        s + "/oauth/token",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"scopes_supported":                      []string{"mcp"},
	}
}
