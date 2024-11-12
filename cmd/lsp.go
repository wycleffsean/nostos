/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// lspCmd represents the lsp command
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Language Server",
	Long: ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("lsp called")
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lspCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lspCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
