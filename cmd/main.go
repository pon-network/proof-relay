package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pon-relay",
	Short: "pon-relay ",
	Long:  `https://pon-relay.com`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pon-relay")
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
