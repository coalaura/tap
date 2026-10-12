package main

func GetConfigs() []*UpgradeConfig {
	return []*UpgradeConfig{
		// GoLang
		{
			Repository: "golang/go",
			Prefix:     "go",

			Binary: "go",
			Path:   GetGoBinaryPath(),
			Args:   []string{"version"},

			Installer: InstallGo,
			Resolver:  FetchLatestGoVersion,
		},

		// Zig
		{
			Repository: "ziglang/zig",
			Prefix:     "",

			Binary: "zig",
			Path:   GetZigBinaryPath(),
			Args:   []string{"version"},

			Installer: InstallZig,
			Resolver:  FetchLatestZigVersion,
		},

		// UPX
		{
			Repository: "upx/upx",
			Prefix:     "v",
			Releases:   true,

			Binary: "upx",
			Path:   GetLocalBinaryPath("upx"),
			Args:   []string{"--version"},

			Installer: InstallUPX,
		},

		// Starship
		{
			Repository: "starship/starship",
			Prefix:     "v",
			Releases:   true,

			Binary: "starship",
			Path:   GetLocalBinaryPath("starship"),
			Args:   []string{"--version"},

			Installer: InstallStarship,
		},

		// Bun
		{
			Repository: "oven-sh/bun",
			Prefix:     "bun-v",
			Releases:   true,

			Binary: "bun",
			Path:   GetBunBinaryPath(),
			Args:   []string{"--version"},

			Installer: InstallBun,
		},

		// Builder
		{
			Repository: "coalaura/builder",
			Prefix:     "v",
			Releases:   true,

			Binary: "builder",
			Path:   GetLocalBinaryPath("builder"),
			Args:   []string{"--version"},
		},

		// Actup
		{
			Repository: "coalaura/actup",
			Prefix:     "v",
			Releases:   true,

			Binary: "actup",
			Path:   GetLocalBinaryPath("actup"),
			Args:   []string{"--version"},
		},

		// License
		{
			Repository: "coalaura/license",
			Prefix:     "v",
			Releases:   true,

			Binary: "license",
			Path:   GetLocalBinaryPath("license"),
			Args:   []string{"--version"},
		},

		// MkSVC
		{
			Repository: "coalaura/mksvc",
			Prefix:     "v",
			Releases:   true,

			Binary:    "mksvc",
			Path:      GetLocalBinaryPath("mksvc"),
			Args:      []string{"--version"},
			AssetName: VersionedGitHubAssetName,
		},

		// Vet
		{
			Repository: "coalaura/vet",
			Prefix:     "v",
			Releases:   true,

			Binary: "vet",
			Path:   GetLocalBinaryPath("vet"),
			Args:   []string{"--version"},
		},

		// Time
		{
			Repository: "coalaura/time",
			Prefix:     "v",
			Releases:   true,

			Binary:    "time",
			Path:      GetLocalBinaryPath("time"),
			Args:      []string{"--version"},
			AssetName: VersionedGitHubAssetName,
		},

		// Wtf
		{
			Repository: "coalaura/wtf",
			Prefix:     "v",
			Releases:   true,

			Binary:    "wtf",
			Path:      GetLocalBinaryPath("wtf"),
			Args:      []string{"--version"},
			AssetName: VersionedGitHubAssetName,
		},

		// Coreutils
		{
			Repository: "uutils/coreutils",
			Prefix:     "",
			Releases:   true,

			Binary: "coreutils",
			Path:   GetLocalBinaryPath("coreutils"),
			Args:   []string{"--version"},

			Installer: InstallCoreutils,
		},
	}
}
