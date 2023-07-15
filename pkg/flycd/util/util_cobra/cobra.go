package util_cobra

import "github.com/spf13/cobra"

type FlagStructIfc interface {
	Init(cmd *cobra.Command)
}

func CreateCmd(flags FlagStructIfc, fnSetupCommand func() *cobra.Command) *cobra.Command {
	cmd := fnSetupCommand()
	flags.Init(cmd)
	return cmd
}
