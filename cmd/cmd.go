package commands

import (
	"github.com/spf13/cobra"
)

func init() {

	rootCmd.Flags().BoolP("version", "v", false, "version for MTA")

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(createMtaCmd)
	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(deleteFileCmd)
	rootCmd.AddCommand(existCmd)
	addCmd.AddCommand(addModuleCmd, addResourceCmd)
	getCmd.AddCommand(getModulesCmd, getResourcesCmd)

	rootCmd.AddCommand(resolveMtaCmd)
	updateCmd.AddCommand(updateModuleCmd, updateResourceCmd, updateBuildParametersCmd)

}

// The parent command adds any artifacts.
var addCmd = &cobra.Command{
	Use:    "add",
	Short:  "Add artifacts",
	Long:   "Add artifacts",
	Hidden: true,
	Run:    nil,
}

// The parent command gets any artifacts.
var getCmd = &cobra.Command{
	Use:    "get",
	Short:  "Get artifacts",
	Long:   "Get artifacts",
	Hidden: true,
	Run:    nil,
}

// The parent command updates the artifacts.
var updateCmd = &cobra.Command{
	Use:    "update",
	Short:  "Update artifact",
	Long:   "Update artifact",
	Hidden: true,
	Run:    nil,
}
