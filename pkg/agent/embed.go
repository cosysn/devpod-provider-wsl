//go:build embed
// +build embed

package agent

import "embed"

//go:embed agent-linux
var Agent embed.FS

func GetAgent() ([]byte, error) {
	return Agent.ReadFile("agent-linux")
}

