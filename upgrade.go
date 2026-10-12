package main

import (
	"errors"

	"github.com/coalaura/plain/minimal"
	"github.com/coalaura/semver"
)

type Installer func(semver.SemVer) error
type VersionResolver func() (semver.SemVer, error)
type AssetNameResolver func(binary, version string) string

type UpgradeConfig struct {
	Name       string
	Repository string
	Prefix     string
	Releases   bool

	Binary    string
	Path      string
	Args      []string
	AssetName AssetNameResolver

	Installer Installer
	Resolver  VersionResolver
}

func (u *UpgradeConfig) GetName() string {
	if u.Name != "" {
		return u.Name
	}

	return u.Binary
}

func (u *UpgradeConfig) Install(ver semver.SemVer) error {
	if u.Installer != nil {
		return u.Installer(ver)
	}

	version := ver.String()
	assetResolver := u.AssetName

	if assetResolver == nil {
		assetResolver = DefaultGitHubAssetName
	}

	asset := assetResolver(u.Binary, version)
	tag := u.Prefix + version

	return InstallGitHubExecutable(u.Repository, tag, asset, u.Path, ver, u.Args)
}

func (u *UpgradeConfig) Upgrade() error {
	log.Infof("Checking %s version...\n", u.GetName())

	remote, err := u.FetchLatestVersion()
	if err != nil {
		return err
	}

	local, err := u.ResolveCurrentVersion()
	if err != nil {
		return err
	}

	if !local.Equal(semver.NewEmptySemVer()) && !remote.HigherThan(local) {
		log.Subf("Already up-to-date (%s == %s)\n", remote, local)

		return nil
	}

	log.Subf("New version found (%s > %s)\n", remote, local)

	log.Infof("Upgrading %s...\n", u.GetName())

	err = u.Install(remote)
	if err != nil {
		return err
	}

	log.Info("Validating upgrade...")

	local, err = u.ResolveCurrentVersion()
	if err != nil {
		log.Writeln(minimal.AnsiError + "failed")

		return err
	}

	if !remote.Equal(local) {
		log.Writeln(minimal.AnsiError + "failed")

		return errors.New("installed version does not match requested version")
	}

	log.Writeln(minimal.AnsiSuccess + "success")

	return nil
}
