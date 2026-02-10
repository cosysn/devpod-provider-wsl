//go:build !embed
// +build !embed

package agent

func GetAgent() ([]byte, error) {
	return nil, nil
}
