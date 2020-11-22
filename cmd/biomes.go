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
	"fmt"
	"log"

	"github.com/danhale-git/craft/internal/biomes"

	"github.com/spf13/cobra"
)

// biomesCmd represents the biomes command
var biomesCmd = &cobra.Command{
	Use: "biomes",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("biomes called")

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

		// Update biome file or return an error
		return biomes.UpdateFile(filePath, key, value)
	},
}

func init() {
	rootCmd.AddCommand(biomesCmd)

	biomesCmd.Flags().StringP("file-path", "f", "", "Path to the server.properties file")

	if err := biomesCmd.MarkFlagRequired("file-path"); err != nil {
		log.Fatal(err)
	}

	biomesCmd.Flags().StringP("key", "k", "", "Key of server properties item to operate on")

	if err := biomesCmd.MarkFlagRequired("key"); err != nil {
		log.Fatal(err)
	}

	biomesCmd.Flags().StringP("value", "v", "", "Value to set")

	if err := biomesCmd.MarkFlagRequired("value"); err != nil {
		log.Fatal(err)
	}
}
