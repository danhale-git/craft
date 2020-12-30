package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/props"
	"github.com/spf13/cobra"
)

// propsCmd represents the props command
var propsCmd = &cobra.Command{
	Use:       "props",
	Short:     "(not currently in use)",
	ValidArgs: []string{"set"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.OnlyValidArgs(cmd, args); err != nil {
			return err
		}

		if err := cobra.RangeArgs(1, 1)(cmd, args); err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flag values
		filePath, err := cmd.Flags().GetString("file-path")
		if err != nil {
			return err
		}

		key, err := cmd.Flags().GetString("key")
		if err != nil {
			return err
		}

		value, err := cmd.Flags().GetString("value")
		if err != nil {
			return err
		}

		// Set the property or return an error
		return props.SetProperty(filePath, key, value)
	},
}

func init() {
	rootCmd.AddCommand(propsCmd)
	propsCmd.Flags().StringP("file-path", "f", "", "Path to the server.properties file")

	if err := propsCmd.MarkFlagRequired("file-path"); err != nil {
		log.Fatal(err)
	}

	propsCmd.Flags().StringP("key", "k", "", "Key of server properties item to operate on")

	if err := propsCmd.MarkFlagRequired("key"); err != nil {
		log.Fatal(err)
	}

	propsCmd.Flags().StringP("value", "v", "", "Value to set")

	if err := propsCmd.MarkFlagRequired("value"); err != nil {
		log.Fatal(err)
	}
}
