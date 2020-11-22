/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"log"

	"github.com/danhale-git/craft/internal/props"
	"github.com/spf13/cobra"
)

// propsCmd represents the props command
var propsCmd = &cobra.Command{
	Use:       "props",
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

	rootCmd.AddCommand(propsCmd)
	propsCmd.Flags().StringP("key", "k", "", "Key of server properties item to operate on")

	if err := propsCmd.MarkFlagRequired("key"); err != nil {
		log.Fatal(err)
	}

	rootCmd.AddCommand(propsCmd)
	propsCmd.Flags().StringP("value", "v", "", "Value to set")

	if err := propsCmd.MarkFlagRequired("value"); err != nil {
		log.Fatal(err)
	}
}
