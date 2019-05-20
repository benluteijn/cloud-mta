package commands

import (
	"encoding/json"
	"fmt"
	"github.com/SAP/cloud-mta/internal/logs"
	"github.com/SAP/cloud-mta/mta"
	"github.com/spf13/cobra"
)

var addModuleMtaCmdPath string
var addModuleCmdData string
var addModuleCmdHashcode int
var getModulesCmdPath string

func init() {
	// set flags of commands
	addModuleCmd.Flags().StringVarP(&addModuleMtaCmdPath, "path", "p", "",
		"the path to the yaml file")
	addModuleCmd.Flags().StringVarP(&addModuleCmdData, "data", "d", "",
		"data in JSON format")
	addModuleCmd.Flags().IntVarP(&addModuleCmdHashcode, "hashcode", "c", 0,
		"data hashcode")
	getModulesCmd.Flags().StringVarP(&getModulesCmdPath, "path", "p", "",
		"the path to the yaml file")
}

// addModuleCmd Add new module
var addModuleCmd = &cobra.Command{
	Use:   "module",
	Short: "Add new module",
	Long:  "Add new module",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logs.Logger.Info("add new module")
		err := mta.ModifyMta(addModuleMtaCmdPath, func() error {
			return mta.AddModule(addModuleMtaCmdPath, addModuleCmdData, mta.Marshal)
		}, addModuleCmdHashcode, false)
		if err != nil {
			logs.Logger.Error(err)
		}
		return err
	},
	Hidden:        true,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// getModulesCmd Get all modules
var getModulesCmd = &cobra.Command{
	Use:   "modules",
	Short: "Get all modules",
	Long:  "Get all modules",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logs.Logger.Info("get modules")
		modules, err := mta.GetModules(getModulesCmdPath)
		if err != nil {
			logs.Logger.Error(err)
		}
		if modules != nil {
			output, rerr := json.Marshal(modules)
			if rerr != nil {
				logs.Logger.Error(rerr)
				return rerr
			}
			fmt.Print(string(output))
		}
		return err
	},
	Hidden:        true,
	SilenceUsage:  true,
	SilenceErrors: true,
}