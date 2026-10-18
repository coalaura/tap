package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Packages map[string]variantRecord `yaml:"packages,omitempty"`
}

func ConfigPath() (string, error) {
	home, err := UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "tap.yml"), nil
}

func ReadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}

	if err != nil {
		return Config{}, err
	}

	defer file.Close()

	var config Config

	decoder := yaml.NewDecoder(file, yaml.Strict())

	err = decoder.Decode(&config)
	if errors.Is(err, io.EOF) {
		return Config{}, nil
	}

	if err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}

	var extra any

	err = decoder.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("%s: expected a single YAML document", path)
	}

	return config, nil
}

func WriteConfig(config Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)

	err = os.MkdirAll(directory, 0755)
	if err != nil {
		return err
	}

	err = ChownToInvokingUser(directory)
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(directory, ".tap-*")
	if err != nil {
		return err
	}

	defer os.Remove(file.Name())

	encodeErr := yaml.NewEncoder(file).Encode(config)

	syncErr := file.Sync()
	closeErr := file.Close()

	err = errors.Join(encodeErr, syncErr, closeErr)
	if err != nil {
		return err
	}

	err = os.Chmod(file.Name(), 0644)
	if err != nil {
		return err
	}

	err = ChownToInvokingUser(file.Name())
	if err != nil {
		return err
	}

	return ReplaceFile(file.Name(), path)
}
