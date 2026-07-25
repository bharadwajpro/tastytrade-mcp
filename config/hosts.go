package config

type Environment string

const (
	EnvProduction Environment = "production"
	EnvSandbox    Environment = "sandbox"
)

const (
	ProdAPI       = "https://api.tastyworks.com"
	SandboxAPI    = "https://api.cert.tastyworks.com"
	ProdStream    = "wss://streamer.tastyworks.com"
	SandboxStream = "wss://streamer.cert.tastyworks.com"
)

// resolveHost: override beats env switch; sandbox env => sandbox host; else prod.
func resolveHost(override string, env Environment, prod, sandbox string) string {
	if override != "" {
		return override
	}
	if env == EnvSandbox {
		return sandbox
	}
	return prod
}
