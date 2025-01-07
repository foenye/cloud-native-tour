package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func main() {
	var version bool
	cmd := &cobra.Command{
		Use:   "root [sub]",
		Short: "root command",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Inside rootCmd Run with args: %v\n", args)
			if version {
				fmt.Println("Version:1.0")
			}
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&version, "version", "v", false, "Print version information and quit")
	_ = cmd.Execute()
}
