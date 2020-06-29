package commands

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack"
	pubbldpkg "github.com/buildpacks/pack/buildpackage"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/logging"
)

// PackageBuildpackFlags define flags provided to the PackageBuildpack command
type PackageBuildpackFlags struct {
	PackageTomlPath string
	Format          string
	Publish         bool
	NoPull          bool
}

// BuildpackPackager packages buildpacks
type BuildpackPackager interface {
	PackageBuildpack(ctx context.Context, options pack.PackageBuildpackOptions) error
}

// PackageConfigReader reads PackageBuildpack configs
type PackageConfigReader interface {
	Read(path string) (pubbldpkg.Config, error)
}

// PackageBuildpack packages (a) buildpack(s) into OCI format, baed on a package config
func PackageBuildpack(logger logging.Logger, client BuildpackPackager, packageConfigReader PackageConfigReader) *cobra.Command {
	var flags PackageBuildpackFlags
	cmd := &cobra.Command{
		Use:   `package-buildpack <name> --config <package-config-path>`,
		Short: "Package buildpack in OCI format.",
		Args:  cobra.ExactValidArgs(1),
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			if err := flags.validate(); err != nil {
				return err
			}

			if cmd.Flags().Changed("package-config") {
				logger.Warn("Flag --package-config has been deprecated, please use --config instead")
			}

			config, err := packageConfigReader.Read(flags.PackageTomlPath)
			if err != nil {
				return errors.Wrap(err, "reading config")
			}

			name := args[0]
			if err := client.PackageBuildpack(cmd.Context(), pack.PackageBuildpackOptions{
				Name:    name,
				Format:  flags.Format,
				Config:  config,
				Publish: flags.Publish,
				NoPull:  flags.NoPull,
			}); err != nil {
				return err
			}

			action := "created"
			if flags.Publish {
				action = "published"
			}

			logger.Infof("Successfully %s package %s", action, style.Symbol(name))
			return nil
		}),
	}
	cmd.Flags().StringVarP(&flags.PackageTomlPath, "package-config", "p", "", "Path to package TOML config (required)")
	cmd.Flags().StringVarP(&flags.PackageTomlPath, "config", "c", "", "Path to package TOML config (required)")

	// TODO: Mark config required and remove package-config after release of pack v0.12: https://github.com/buildpacks/pack/issues/694
	// cmd.MarkFlagRequired("config")
	cmd.Flags().MarkHidden("package-config")

	cmd.Flags().StringVarP(&flags.Format, "format", "f", "", `Format to save package as ("image" or "file")`)
	cmd.Flags().BoolVar(&flags.Publish, "publish", false, `Publish to registry (applies to "--image" only)`)
	cmd.Flags().BoolVar(&flags.NoPull, "no-pull", false, "Skip pulling packages before use")
	AddHelpFlag(cmd, "package-buildpack")

	return cmd
}

func (p PackageBuildpackFlags) validate() error {
	if p.Publish && p.NoPull {
		return errors.Errorf("The --publish and --no-pull flags cannot be used together. The --publish flag requires the use of remote images.")
	}

	if p.PackageTomlPath == "" {
		return errors.Errorf("Please provide a package config path, using --config")
	}
	return nil
}
