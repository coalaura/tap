package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/coalaura/semver"
	"github.com/urfave/cli/v3"
)

type packageUpdate struct {
	Package   *Package
	Current   semver.SemVer
	Target    semver.SemVer
	Reinstall bool
	Err       error
}

func (update *packageUpdate) needed() bool {
	return update.Err == nil && (update.Reinstall || update.Target.HigherThan(update.Current))
}

func (update *packageUpdate) check() error {
	pkg := update.Package

	if len(pkg.Definition.Variants) != 0 {
		variant, err := pkg.InstalledVariant()
		if err != nil {
			return err
		}

		if variant == "" {
			return errors.New("installed variant is unknown; select it explicitly with tap get package:variant")
		}

		pkg, err = ResolvePackage(pkg.Definition, variant)
		if err != nil {
			return err
		}

		update.Package = pkg
	}

	var err error

	update.Current, err = pkg.ResolveCurrentVersion()
	if err != nil {
		return err
	}

	if update.Current.Equal(semver.NewEmptySemVer()) {
		return fmt.Errorf("installed version is unknown or installation is incomplete; use tap get %s to repair it", pkg.Name())
	}

	update.Target, err = pkg.FetchLatestVersion()
	return err
}

func updatePackages(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return errors.New("usage: tap up [--dry] [--yes]")
	}

	log.Infoln("collecting installed packages...")

	checks, err := collectPackageUpdates(ctx)
	if err != nil {
		return err
	}

	plan, failures := planPackageUpdates(checks)

	for _, failure := range failures {
		log.Warnf("skipping %v\n", failure)
	}

	if len(plan) == 0 {
		switch {
		case len(failures) != 0:
			log.Infoln("no updates available for the packages checked successfully")
		case len(checks) == 0:
			log.Infoln("no installed packages found")
		default:
			log.Infoln("all installed packages are up to date")
		}

		return errors.Join(failures...)
	}

	err = showUpdatePlan(command.Root().Writer, plan)
	if err != nil {
		return errors.Join(errors.Join(failures...), err)
	}

	if command.Bool("dry") {
		return errors.Join(failures...)
	}

	if !command.Bool("yes") {
		confirmed, err := confirmPackageUpdates(command.Root().Reader, command.Root().Writer)
		if err != nil {
			return errors.Join(errors.Join(failures...), err)
		}

		if !confirmed {
			log.Infoln("update cancelled")

			return errors.Join(failures...)
		}
	}

	err = applyPackageUpdates(ctx, plan)

	return errors.Join(errors.Join(failures...), err)
}

func installPackageVersion(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 2 {
		return errors.New("usage: tap install-version <package[:variant]> <version>")
	}

	packages, err := SelectPackages(command.Args().Slice()[:1])
	if err != nil {
		return err
	}

	version, err := semver.ParseSemVer(command.Args().Get(1), false)
	if err != nil {
		return err
	}

	return installResolvedPackage(ctx, packages[0], version)
}

func collectPackageUpdates(ctx context.Context) ([]packageUpdate, error) {
	packages, err := SelectPackages(nil)
	if err != nil {
		return nil, err
	}

	checks := make([]packageUpdate, 0, len(packages))

	for _, pkg := range packages {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		installed, err := pkg.IsInstalled()
		if err == nil && !installed {
			continue
		}

		check := packageUpdate{
			Package: pkg,
			Current: semver.Invalid,
			Target:  semver.Invalid,
			Err:     err,
		}

		if err == nil {
			log.Infof("checking %s...\n", pkg.Definition.Name)

			check.Err = check.check()
		}

		checks = append(checks, check)
	}

	return checks, ctx.Err()
}

func planPackageUpdates(checks []packageUpdate) ([]packageUpdate, []error) {
	var (
		goUpdate   *packageUpdate
		paceUpdate *packageUpdate
	)

	for index := range checks {
		check := &checks[index]

		switch check.Package.Definition.Source {
		case "go":
			goUpdate = check
		case "pace":
			paceUpdate = check
		}
	}

	planPaceUpdate(goUpdate, paceUpdate)

	plan := make([]packageUpdate, 0, len(checks))
	failures := make([]error, 0, len(checks))

	// Go must finish before PACE validates its required toolchain.
	if goUpdate != nil && goUpdate.needed() {
		plan = append(plan, *goUpdate)
	}

	for index := range checks {
		check := &checks[index]
		if check.Err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", check.Package.Definition.Name, check.Err))

			continue
		}

		if check != goUpdate && check.needed() {
			plan = append(plan, *check)
		}
	}

	return plan, failures
}

func planPaceUpdate(goUpdate, paceUpdate *packageUpdate) {
	if paceUpdate == nil {
		return
	}

	if goUpdate != nil && goUpdate.needed() {
		switch {
		case paceUpdate.Err != nil:
			goUpdate.Err = errors.New("cannot update Go while the installed PACE version could not be checked")
		case !goUpdate.Target.Equal(paceUpdate.Target) && !goUpdate.Target.Equal(paceUpdate.Current):
			goUpdate.Err = fmt.Errorf("go %s has no matching installed or latest PACE release", goUpdate.Target)
		case paceUpdate.Current.HigherThan(goUpdate.Target):
			goUpdate.Err = errors.New("updating Go would require downgrading PACE")
		default:
			// Replacing Go on Linux removes PACE even when its version stays the same.
			if runtime.GOOS == "linux" || goUpdate.Target.HigherThan(paceUpdate.Current) {
				paceUpdate.Target = goUpdate.Target
				paceUpdate.Reinstall = paceUpdate.Target.Equal(paceUpdate.Current)
			}
		}
	}

	if !paceUpdate.needed() {
		return
	}

	if goUpdate == nil || goUpdate.Current.IsInvalid() {
		paceUpdate.Err = errors.New("cannot update PACE without a known installed Go version")

		return
	}

	goVersion := goUpdate.Current

	if goUpdate.needed() {
		goVersion = goUpdate.Target
	}

	if !paceUpdate.Target.Equal(goVersion) {
		paceUpdate.Err = fmt.Errorf("PACE %s requires Go %s, but the planned Go version is %s", paceUpdate.Target, paceUpdate.Target, goVersion)
	}
}

func showUpdatePlan(writer io.Writer, plan []packageUpdate) error {
	table := tabwriter.NewWriter(writer, 0, 4, 2, ' ', 0)

	fmt.Fprintln(table, "package\tcurrent\ttarget\taction")

	for index := range plan {
		update := &plan[index]
		action := "update"

		if update.Reinstall {
			action = "reinstall after Go"
		}

		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", update.Package.Name(), update.Current, update.Target, action)
	}

	return table.Flush()
}

func confirmPackageUpdates(reader io.Reader, writer io.Writer) (bool, error) {
	_, err := fmt.Fprint(writer, "Apply this update plan? [y/N] ")
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return false, scanner.Err()
	}

	answer := strings.TrimSpace(scanner.Text())

	return strings.EqualFold(answer, "y") || strings.EqualFold(answer, "yes"), nil
}

func applyPackageUpdates(ctx context.Context, plan []packageUpdate) error {
	failures := make([]error, 0, len(plan))

	var (
		updated  int
		goFailed bool
	)

	for index := range plan {
		if ctx.Err() != nil {
			return errors.Join(errors.Join(failures...), ctx.Err())
		}

		update := &plan[index]
		pkg := update.Package

		var err error

		if pkg.Definition.Source == "pace" && goFailed {
			err = errors.New("skipped because the Go update failed")
		} else {
			err = installResolvedPackage(ctx, pkg, update.Target)
		}

		if err != nil {
			failure := fmt.Errorf("%s: %w", pkg.Name(), err)
			failures = append(failures, failure)

			log.Warnf("%v\n", failure)

			if pkg.Definition.Source == "go" {
				goFailed = true
			}

			continue
		}

		updated++
	}

	log.Infof("finished: %d updated, %d failed\n", updated, len(failures))

	return errors.Join(failures...)
}

func installResolvedPackage(ctx context.Context, pkg *Package, version semver.SemVer) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err := pkg.checkPermissions(false)
	if err == nil {
		err = pkg.installVersion(version)
	}

	if err == nil {
		return nil
	}

	return retryElevated(ctx, []string{"install-version", pkg.Name(), version.String()}, err)
}
