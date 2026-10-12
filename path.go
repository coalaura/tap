package main

var (
	userHomeDir string
)

func UserHomeDir() (string, error) {
	if userHomeDir == "" {
		dir, err := ResolveUserHomeDir()
		if err != nil {
			return "", err
		}

		userHomeDir = dir
	}

	return userHomeDir, nil
}
