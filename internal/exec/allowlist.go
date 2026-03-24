package exec

// defaultAllowedBins is the set of binaries that RealRunner permits by default.
var defaultAllowedBins = map[string]bool{
	"docker":         true,
	"apt-get":        true,
	"wsl.exe":        true,
	"netsh":          true,
	"powershell.exe": true,
	"ollama":         true,
	"curl":           true,
	"lsof":           true,
	"nvidia-smi":     true,
}

// allowedBins returns the runner's AllowedBins if set, otherwise the default.
func (r *RealRunner) allowedBins() map[string]bool {
	if r.AllowedBins != nil {
		return r.AllowedBins
	}
	return defaultAllowedBins
}
