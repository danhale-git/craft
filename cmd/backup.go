package cmd

import (
	"strings"

	"github.com/danhale-git/craft/craft"
	"github.com/danhale-git/craft/internal/docker"
	"github.com/danhale-git/craft/internal/logger"
	"github.com/spf13/cobra"
)

// NewBackupCmd returns the backup command which saves a local backup of the server and world.
func NewBackupCmd() *cobra.Command {
	backupCmd := &cobra.Command{
		Use:   "backup <server names...>",
		Short: "Take a backup",
		Long: `
Save the current world and server.properties to a zip file in the backup directory.
If two backups are taken in the same minute, the second will overwrite the first.
Backups are saved to a default directory under the user's home directory.
The backed up world is usually a few seconds behind the world state at the time of backup.
Use the trim and skip-trim-file-removal-check flags with linux cron or windows task scheduler to automate backups.`,
		Example: `craft backup myserver
craft backup myserver -l

Linux cron (hourly):
0 * * * * ~/craft_backups/backup.sh
	
	#!/usr/bin/env bash
	~/go/bin/craft backup myserver myotherserver \ # path to craft executable and one or more servers
	--skip-trim-file-removal-check --trim 3 \ # skip cmdline prompts and delete all except 3 newest files
	--log ~/craft_backups/backup.log --log-level info # log to file with log level info
`,
		Args: func(cmd *cobra.Command, args []string) error {
			return cobra.MinimumNArgs(1)(cmd, args)
		},
		Run: backupCommand,
	}

	backupCmd.Flags().IntP("trim", "t", 0,
		"Delete the oldest backup files, leaving the given count of newest files in place.")

	backupCmd.Flags().BoolP("list", "l", false,
		"List backup files and take no other action.")

	backupCmd.Flags().Bool("skip-trim-file-removal-check", false,
		"Don't prompt the user before removing files. Useful for automating backups.")

	return backupCmd
}

func backupCommand(cmd *cobra.Command, args []string) {
	trim, err := cmd.Flags().GetInt("trim")
	if err != nil {
		logger.Panic(err)
	}

	skip, err := cmd.Flags().GetBool("skip-trim-file-removal-check")
	if err != nil {
		logger.Panic(err)
	}

	created := make([]string, 0)
	deleted := make([]string, 0)

	for _, name := range args {
		c := docker.GetContainerOrExit(name)

		// Take a new backup
		name, err := craft.CopyBackup(c)
		if err != nil {
			logger.Error.Fatalf("%s: taking backup: %s", c.ContainerName, err)
		}

		created = append(created, name)

		if trim > 0 {
			del, err := craft.TrimBackups(c.ContainerName, trim, skip)
			if err != nil {
				logger.Error.Printf("%s: trimming old backup files: %s", c.ContainerName, err)
			}

			deleted = append(deleted, del...)
		}
	}

	if len(created) > 0 {
		logger.Info.Println("created:", strings.Join(created, " "))
	}

	if len(deleted) > 0 {
		logger.Info.Println("deleted:", strings.Join(deleted, " "))
	}
}
