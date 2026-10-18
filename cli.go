package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	cli.RootCommandHelpTemplate = helpTemplate
	cli.CommandHelpTemplate = helpTemplate
	cli.SubcommandHelpTemplate = helpTemplate
	cli.UsageCommandHelp = "show help"
	cli.ShowCommandHelp = showCommandHelp

	command := &cli.Command{
		Name:         "tap",
		Usage:        "a small package manager",
		Version:      Version,
		Action:       showHelp,
		Before:       checkPath,
		OnUsageError: usageError,
		Commands: []*cli.Command{
			{
				Name:   "tap",
				Usage:  "update tap",
				Action: updateTap,
			},
			{
				Name:      "get",
				Usage:     "install or update packages",
				ArgsUsage: "<package[:variant]>...",
				Action:    getPackages,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry",
						Usage: "show the version that would be installed without installing",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "reinstall without checking the installed version",
					},
				},
			},
			{
				Name:      "rm",
				Usage:     "remove packages",
				ArgsUsage: "<package[:variant]>...",
				Action:    removePackages,
			},
			{
				Name:   "ls",
				Usage:  "list installed packages",
				Action: listPackages,
			},
			{
				Name:      "find",
				Usage:     "search packages (default: all)",
				ArgsUsage: "[query]",
				Action:    findPackages,
			},
			{
				Name:      "info",
				Usage:     "show package details",
				ArgsUsage: "<package[:variant]>...",
				Action:    packageInfo,
			},
			{
				Name:            "help",
				Aliases:         []string{"h"},
				Usage:           "show help",
				ArgsUsage:       "[command]",
				HideHelpCommand: true,
				Action:          helpCommand,
			},
		},
	}

	for _, subcommand := range command.Commands {
		subcommand.OnUsageError = usageError
	}

	return command
}

func showHelp(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return fmt.Errorf("unknown command %q; see tap --help", command.Args().First())
	}

	return cli.ShowAppHelp(command)
}

func showCommandHelp(ctx context.Context, command *cli.Command, name string) error {
	if command.Command(name) == nil {
		return fmt.Errorf("unknown command %q; see %s --help", name, command.FullName())
	}

	return cli.DefaultShowCommandHelp(ctx, command, name)
}

func helpCommand(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return showCommandHelp(ctx, command.Root(), command.Args().First())
	}

	return cli.ShowAppHelp(command)
}

// Use a command hook so --version stays parseable during self-updates.
func checkPath(ctx context.Context, command *cli.Command) (context.Context, error) {
	WarnMissingPath()

	return ctx, nil
}

func usageError(ctx context.Context, command *cli.Command, err error, subcommand bool) error {
	return fmt.Errorf("%w; see %s --help", err, command.FullName())
}

func updateTap(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() != 0 {
		return errors.New("usage: tap tap")
	}

	err := UpdateTap()

	return retryElevated(ctx, []string{"tap"}, err)
}

func getPackages(ctx context.Context, command *cli.Command) error {
	force := command.Bool("force")
	dry := command.Bool("dry")

	return applyPackages(ctx, command, func(pkg *Package) error {
		err := pkg.Get(force, dry)

		if dry {
			return err
		}

		// Finish a partially completed install even if its primary binary was already replaced.
		return retryElevated(ctx, []string{"get", "--force", pkg.Name()}, err)
	})
}

func removePackages(ctx context.Context, command *cli.Command) error {
	return applyPackages(ctx, command, func(pkg *Package) error {
		err := pkg.Remove()

		return retryElevated(ctx, []string{"rm", pkg.Name()}, err)
	})
}

func applyPackages(ctx context.Context, command *cli.Command, action func(*Package) error) error {
	if command.Args().Len() == 0 {
		return fmt.Errorf("package required; see tap %s --help", command.Name)
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
			failures = append(failures, fmt.Errorf("%s: %w", pkg.Name(), err))
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

	fmt.Fprintln(writer, "package\tversion\tpath")

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

		name := pkg.Definition.Name

		if len(pkg.Definition.Variants) != 0 {
			variant, err := pkg.InstalledVariant()
			if err != nil {
				failures = append(failures, fmt.Errorf("%s: %w", name, err))

				continue
			}

			if variant == "" {
				variant = "unknown"
			}

			name += ":" + variant
		}

		version, err := pkg.ResolveCurrentVersion()
		if err != nil {
			fmt.Fprintf(writer, "%s\tunknown\t%s\n", name, pkg.Path)

			failures = append(failures, fmt.Errorf("%s: %w", pkg.Definition.Name, err))

			continue
		}

		fmt.Fprintf(writer, "%s\t%s\t%s\n", name, version, pkg.Path)
	}

	return errors.Join(errors.Join(failures...), writer.Flush())
}

func findPackages(ctx context.Context, command *cli.Command) error {
	query := strings.ToLower(strings.Join(command.Args().Slice(), " "))

	writer := tabwriter.NewWriter(command.Root().Writer, 0, 4, 2, ' ', 0)

	fmt.Fprintln(writer, "package\tdescription\tsupported")

	for index := range packageCatalog {
		definition := &packageCatalog[index]
		_, supported := definition.Platforms[runtime.GOOS+"-"+runtime.GOARCH]

		if strings.Contains(definition.Name, query) || strings.Contains(strings.ToLower(definition.Description), query) {
			fmt.Fprintf(writer, "%s\t%s\t%t\n", definition.Name, strings.ToLower(definition.Description), supported)
		}

		for _, variant := range slices.Sorted(maps.Keys(definition.Variants)) {
			name := definition.Name + ":" + variant
			if !strings.Contains(name, query) && !strings.Contains(strings.ToLower(definition.Description), query) {
				continue
			}

			fmt.Fprintf(writer, "%s\t%s\t%t\n", name, strings.ToLower(definition.Description), supported)
		}
	}

	return writer.Flush()
}

func packageInfo(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() == 0 {
		return errors.New("package required; see tap info --help")
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

		fmt.Fprintf(writer, "%s - %s\nsource: %s\n", pkg.Name(), strings.ToLower(definition.Description), definition.Source)

		if len(definition.Variants) != 0 {
			fmt.Fprintf(writer, "variants: %s (default: %s)\n", strings.Join(slices.Sorted(maps.Keys(definition.Variants)), ", "), definition.DefaultVariant)
		}

		if definition.Repository != "" {
			fmt.Fprintf(writer, "repo: https://github.com/%s\n", definition.Repository)
		}

		fmt.Fprintf(writer, "path: %s\n", pkg.Path)

		installed, err := pkg.IsInstalled()
		if err != nil {
			return fmt.Errorf("%s: %w", definition.Name, err)
		}

		if !installed {
			fmt.Fprintln(writer, "installed: no")
		} else {
			if len(definition.Variants) != 0 {
				variant, err := pkg.InstalledVariant()
				if err != nil {
					return fmt.Errorf("%s: %w", definition.Name, err)
				}

				if variant == "" {
					variant = "unknown"
				}

				fmt.Fprintf(writer, "installed variant: %s\n", variant)
			}

			version, err := pkg.ResolveCurrentVersion()
			if err != nil {
				return fmt.Errorf("%s: %w", definition.Name, err)
			}

			fmt.Fprintf(writer, "installed: %s\n", version)
		}

		latest, err := pkg.FetchLatestVersion()
		if err != nil {
			return fmt.Errorf("%s: %w", definition.Name, err)
		}

		fmt.Fprintf(writer, "latest: %s\n", latest)
	}

	return nil
}
