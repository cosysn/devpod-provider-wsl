package options

import (
	"fmt"
	"os"
)

var (
	WSL_DISTRO = "WSL_DISTRO"
)

type Options struct {
	WSLDistro string
}

func FromEnv(init, withFolder bool) (*Options, error) {
	retOptions := &Options{}

	var err error
	retOptions.WSLDistro, err = fromEnvOrError(WSL_DISTRO)
	if err != nil {
		return nil, fmt.Errorf("WSL_DISTRO environment variable is required")
	}

	return retOptions, nil
}

func fromEnvOrError(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf(
			"couldn't find option %s in environment, please make sure %s is defined",
			name,
			name,
		)
	}

	return val, nil
}
