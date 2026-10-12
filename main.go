package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/coalaura/plain/minimal"
)

var log = minimal.New()

func main() {
	status := run()

	if status != 0 {
		os.Exit(status)
	}
}

func run() int {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		log.Warnf("unsupported operating system: %s\n", runtime.GOOS)

		return 1
	}

	if runtime.GOARCH != "amd64" {
		log.Warnf("unsupported architecture: %s\n", runtime.GOARCH)

		return 1
	}

	configs := GetConfigs()

	configs, err := FilterConfigs(configs, os.Args[1:])
	if err != nil {
		log.Warnln(err)

		return 1
	}

	failed := false

	for _, cfg := range configs {
		err = cfg.Upgrade()
		if err != nil {
			failed = true

			log.Warnf("%s: %v\n", cfg.GetName(), err)
		}
	}

	if failed {
		log.Warnln("Completed upgrades with errors.")

		return 1
	}

	log.Successln("Completed upgrades.")

	return 0
}

func FilterConfigs(configs []*UpgradeConfig, names []string) ([]*UpgradeConfig, error) {
	if len(names) == 0 {
		return configs, nil
	}

	available := make(map[string]*UpgradeConfig, len(configs))

	for _, cfg := range configs {
		available[strings.ToLower(cfg.GetName())] = cfg
	}

	filtered := make([]*UpgradeConfig, 0, len(names))
	added := make(map[string]struct{}, len(names))

	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))

		cfg, ok := available[name]
		if !ok {
			return nil, fmt.Errorf("unknown tool %q", name)
		}

		if _, ok := added[name]; ok {
			continue
		}

		filtered = append(filtered, cfg)
		added[name] = struct{}{}
	}

	return filtered, nil
}
