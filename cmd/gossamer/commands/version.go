package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "gossamer version",
	Long:  `gossamer version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("gossamer version 0.3.2")
		return nil
	},
}
