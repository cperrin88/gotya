// Package cli contains the gotya CLI commands and subcommands.
package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/cperrin88/gotya/pkg/artifact"
	"github.com/cperrin88/gotya/pkg/errors"
)

// NewArtifactCmd creates a new artifact command.
func NewArtifactCmd() *cobra.Command {
	pkgCmd := &cobra.Command{
		Use:   "artifact",
		Short: "Artifact management commands",
		Long:  "Commands for creating and managing gotya packages",
	}

	// Add subcommands
	pkgCmd.AddCommand(newArtifactCreateCommand())
	pkgCmd.AddCommand(newArtifactVerifyCommand())

	return pkgCmd
}

// newArtifactCreateCommand creates the 'artifact create' command.
func newArtifactCreateCommand() *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new artifact",
		Long: `Create a new gotya artifact from a source directory.
The source directory should contain a 'meta/artifact.json' file and a 'files/' directory.`,
		RunE: func(_ *cobra.Command, _ []string) error { return runCreateArtifact(opts) },
	}
	addCreateFlags(cmd, opts)
	return cmd
}

// newArtifactVerifyCommand creates the 'artifact verify' command.
func newArtifactVerifyCommand() *cobra.Command {
	// Command line flags
	var (
		filePath string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify an artifact file",
		Long: `Verify the integrity of a gotya artifact file.

This command checks the internal consistency of an artifact file, 
including file hashes and metadata structure.`,
		RunE: func(_ *cobra.Command, args []string) error {
			// If file path is provided as an argument, use it (takes precedence over flag)
			if len(args) > 0 {
				filePath = args[0]
			}

			if filePath == "" {
				return fmt.Errorf("missing required argument: file path: %w", errors.ErrValidation)
			}

			// Convert to absolute path
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// Check if file exists
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("artifact file not found: %s: %w", absPath, errors.ErrFileNotFound)
			}

			log.Printf("Verifying artifact: %s\n", absPath)

			verifier := artifact.NewVerifier()
			if err := verifier.VerifyArtifact(context.Background(), nil, absPath); err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			log.Printf("Artifact verified successfully: %s\n", absPath)
			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the artifact file to verify")

	// Mark the file flag as required if not provided as an argument
	_ = cmd.MarkFlagFilename("file", "gotya")

	// Allow passing the file path as a positional argument
	cmd.Args = cobra.MaximumNArgs(1)

	return cmd
}

// createOptions holds flags for the create command.
type createOptions struct {
	sourceDir    string
	outputDir    string
	pkgName      string
	pkgVer       string
	pkgOS        string
	pkgArch      string
	maintainer   string
	description  string
	dependencies []string
	hooks        map[string]string
}

func addCreateFlags(cmd *cobra.Command, o *createOptions) {
	// Add flags with descriptions and defaults
	cmd.Flags().StringVarP(&o.sourceDir, "source", "s", ".", "Source directory containing artifact files (required)")
	cmd.Flags().StringVarP(&o.outputDir, "output", "o", ".", "Output directory for the created artifact")
	cmd.Flags().StringVar(&o.pkgName, "name", "", "Artifact name (required, overrides name in artifact.json)")
	cmd.Flags().StringVar(&o.pkgVer, "version", "0.1.0", "Artifact version (e.g., 1.0.0, overrides version in artifact.json)")
	cmd.Flags().StringVar(&o.pkgOS, "os", runtime.GOOS, "Target operating system")
	cmd.Flags().StringVar(&o.pkgArch, "arch", runtime.GOARCH, "Target architecture")
	cmd.Flags().StringVar(&o.maintainer, "maintainer", "", "Artifact maintainer (name <email>)")
	cmd.Flags().StringVar(&o.description, "description", "", "Artifact description")
	cmd.Flags().StringSliceVar(&o.dependencies, "depends", nil, "Artifact dependencies (comma-separated)")
	cmd.Flags().StringToStringVar(&o.hooks, "hooks", nil, "Artifact hooks in format 'hooks=script.tengo' (comma-separated)")

	// Mark required flags
	must(cmd.MarkFlagRequired("source"))
	must(cmd.MarkFlagRequired("name"))
}

func runCreateArtifact(o *createOptions) error {
	// Parse dependencies
	parsedDeps, err := ParseDependencies(o.dependencies)
	if err != nil {
		return fmt.Errorf("failed to parse dependencies: %w", err)
	}
	packer := artifact.NewPacker(
		o.pkgName,
		o.pkgVer,
		o.pkgOS,
		o.pkgArch,
		o.maintainer,
		o.description,
		parsedDeps,
		o.hooks,
		o.sourceDir,
		o.outputDir,
	)
	outputFile, err := packer.Pack()
	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}
	fmt.Printf("Successfully created artifact: %s\n", outputFile)
	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
