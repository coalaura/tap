package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:    "tap",
		Usage:   "A small package manager for Windows and Linux",
		Version: Version,
		Action:  showHelp,
		Commands: []*cli.Command{
			{
				Name:   "tap",
				Usage:  "Update tap itself to the latest stable release",
				Action: updateTap,
			},
			{
				Name:      "get",
				Usage:     "Install packages or update them to the latest stable version",
				ArgsUsage: "<package>...",
				Action:    getPackages,
			},
			{
				Name:      "rm",
				Usage:     "Remove installed packages",
				ArgsUsage: "<package>...",
				Action:    removePackages,
			},
			{
				Name:   "ls",
				Usage:  "List installed packages from the catalog",
				Action: listPackages,
			},
			{
				Name:      "find",
				Usage:     "Search the catalog by name or description; omit the query to show all",
				ArgsUsage: "[query]",
				Action:    findPackages,
			},
			{
				Name:      "info",
				Usage:     "Show package details and local installation status",
				ArgsUsage: "<package>...",
				Action:    packageInfo,
			},
		},
	}
}

func showHelp(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return fmt.Errorf("unknown command %q; see tap --help", command.Args().First())
	}

	return cli.ShowAppHelp(command)
}

func updateTap(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return errors.New("usage: tap tap")
	}

	return UpdateTap()
}

func getPackages(ctx context.Context, command *cli.Command) error {
	return applyPackages(ctx, command, (*Package).Get)
}

func removePackages(ctx context.Context, command *cli.Command) error {
	return applyPackages(ctx, command, (*Package).Remove)
}

func applyPackages(ctx context.Context, command *cli.Command, action func(*Package) error) error {
	if command.Args().Len() == 0 {
		return fmt.Errorf("usage: tap %s <package> [<package>...]", command.Name)
	}

	packages, err := SelectPackages(command.Args().Slice())
	if err != nil {
		return err
	}

	var failures []error

	for _, pkg := range packages {
		if ctx.Err() != nil {
			return errors.Join(errors.Join(failures...), ctx.Err())
		}

		err = action(pkg)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", pkg.Definition.Name, err))
		}
	}

	return errors.Join(failures...)
}

func listPackages(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return errors.New("usage: tap ls")
	}

	packages, err := SelectPackages(nil)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(command.Root().Writer, 0, 4, 2, ' ', 0)

	fmt.Fprintln(writer, "PACKAGE\tVERSION\tPATH")

	var failures []error

	for _, pkg := range packages {
		installed, err := pkg.IsInstalled()
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", pkg.Definition.Name, err))

			continue
		}

		if !installed {
			continue
		}

		version, err := pkg.ResolveCurrentVersion()
		if err != nil {
			fmt.Fprintf(writer, "%s\tunknown\t%s\n", pkg.Definition.Name, pkg.Path)

			failures = append(failures, fmt.Errorf("%s: %w", pkg.Definition.Name, err))

			continue
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\n", pkg.Definition.Name, version, pkg.Path)
	}

	return errors.Join(errors.Join(failures...), writer.Flush())
}

func findPackages(ctx context.Context, command *cli.Command) error {
	query := strings.ToLower(strings.Join(command.Args().Slice(), " "))

	writer := tabwriter.NewWriter(command.Root().Writer, 0, 4, 2, ' ', 0)

	fmt.Fprintln(writer, "PACKAGE\tDESCRIPTION\tSUPPORTED")

	for index := range packageCatalog {
		definition := &packageCatalog[index]
		if !strings.Contains(definition.Name, query) && !strings.Contains(strings.ToLower(definition.Description), query) {
			continue
		}

		_, supported := definition.Platforms[runtime.GOOS+"-"+runtime.GOARCH]

		fmt.Fprintf(writer, "%s\t%s\t%t\n", definition.Name, definition.Description, supported)
	}

	return writer.Flush()
}

func packageInfo(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() == 0 {
		return errors.New("usage: tap info <package> [<package>...]")
	}

	packages, err := SelectPackages(command.Args().Slice())
	if err != nil {
		return err
	}

	writer := command.Root().Writer

	for index, pkg := range packages {
		if index != 0 {
			fmt.Fprintln(writer)
		}

		definition := pkg.Definition

		fmt.Fprintf(writer, "%s — %s\nSource: %s\n", definition.Name, definition.Description, definition.Source)

		if definition.Repository != "" {
			fmt.Fprintf(writer, "Repository: https://github.com/%s\n", definition.Repository)
		}

		fmt.Fprintf(writer, "Path: %s\n", pkg.Path)

		installed, err := pkg.IsInstalled()
		if err != nil {
			return fmt.Errorf("%s: %w", definition.Name, err)
		}

		if !installed {
			fmt.Fprintln(writer, "Installed: no")

			continue
		}

		version, err := pkg.ResolveCurrentVersion()
		if err != nil {
			return fmt.Errorf("%s: %w", definition.Name, err)
		}

		fmt.Fprintf(writer, "Installed: %s\n", version)
	}

	return nil
}
