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
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/danhale-git/craft/internal/biomes"

	"github.com/spf13/cobra"
)

// biomesCmd represents the biomes command
var biomesCmd = &cobra.Command{
	Use: "biomes",
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

		fi, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			var files []string

			err := filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
				if info.Mode().IsRegular() {
					files = append(files, path)
				}
				return nil
			})

			if err != nil {
				return err
			}

			fmt.Printf("change %d files?\nCtrl-C to abort", len(files))
			bufio.NewReader(os.Stdin).ReadString('\n')

			for _, file := range files {
				err = OperateOnFile(file, key, value, args)
				if err != nil {

					return err
				}
			}

		case mode.IsRegular():
			// Update biome file or return an error
			return OperateOnFile(filePath, key, value, args)
		}

		return nil

	},
}

func OperateOnFile(filePath, key string, value interface{}, args []string) error {
	switch args[0] {
	case "setvalue":
		err := biomes.UpdateValue(filePath, key, value)

		if err != nil {
			switch err.(type) {
			case biomes.KeyNotFoundError:
				fmt.Println(err)
			default:
				return err
			}
		}

		return nil
	case "replaceallblocks":
		err := biomes.UpdateAllSurfaceMaterials(filePath, value.(string))

		if err != nil {
			switch err.(type) {
			case biomes.KeyNotFoundError:
				fmt.Println(err)
			default:
				return err
			}
		}

		return nil
	default:
		return fmt.Errorf("unrecognised argument: '%s'", args[0])
	}
}

func init() {
	rootCmd.AddCommand(biomesCmd)

	biomesCmd.Flags().StringP("file-path", "f", "", "Path to the directory containing biomes.")

	if err := biomesCmd.MarkFlagRequired("file-path"); err != nil {
		log.Fatal(err)
	}

	biomesCmd.Flags().StringP("key", "k", "", "Key of server properties item to operate on")

	biomesCmd.Flags().StringP("value", "v", "", "Value to set")

	if err := biomesCmd.MarkFlagRequired("value"); err != nil {
		log.Fatal(err)
	}
}
