package container

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rootCmd = &cobra.Command{
	Use: "container",
}

var rootCmdInitDone = false

func GetRootCmd() *cobra.Command {
	if !rootCmdInitDone {
		// setup common args
		fs := rootCmd.PersistentFlags()
		fs.String("project", "", "")
		fs.String("region", "", "")
		fs.String("zone", "", "")
		registerConfigHelperCmd(rootCmd)
		rootCmdInitDone = true
	}
	return rootCmd
}

type commonArgs struct {
	project string
	region  string
	zone    string
}

func getCommonFlags(fs *pflag.FlagSet) (*commonArgs, error) {
	project, _ := fs.GetString("project")
	if project == "" {
		return nil, errors.New("--project is required")
	}
	region, _ := fs.GetString("region")
	zone, _ := fs.GetString("zone")
	if region == "" && zone == "" {
		return nil, errors.New("either --project or --zone is required")
	}
	if region != "" && zone != "" {
		return nil, errors.New("either --project or --zone is required")
	}
	return &commonArgs{
		project: project,
		region:  region,
		zone:    zone,
	}, nil
}
