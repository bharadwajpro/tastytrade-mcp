package mcptools

import "fmt"

// resolveAccount picks the effective account.
//
//	0 configured           -> error
//	1 configured           -> that one (arg ignored)
//	>1 configured, no arg   -> error listing choices
//	>1 configured, arg set  -> arg if it's one of them, else error
func resolveAccount(configured []string, arg string) (string, error) {
	switch len(configured) {
	case 0:
		return "", fmt.Errorf("no account configured (set TT_ACCOUNT_NUMBER)")
	case 1:
		return configured[0], nil
	default:
		if arg == "" {
			return "", fmt.Errorf("account argument required; configured: %v", configured)
		}
		for _, a := range configured {
			if a == arg {
				return a, nil
			}
		}
		return "", fmt.Errorf("unknown account %q; configured: %v", arg, configured)
	}
}
