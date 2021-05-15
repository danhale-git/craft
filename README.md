![Go Report Card](https://goreportcard.com/badge/github.com/danhale-git/craft)
![example workflow](https://github.com/danhale-git/craft/actions/workflows/golangci-lint.yml/badge.svg)
![example workflow](https://github.com/danhale-git/craft/actions/workflows/go-test.yaml/badge.svg)
![coverage](https://github.com/danhale-git/craft/actions/workflows/coverage.yaml/badge.svg)

# Craft
Craft is a simple tool for running and managing Bedrock servers.

It's a docker API wrapper which runs a specific container with bedrock installed.

Windows and Linux (tested on Ubuntu 20) are supported.

### Examples

    # Start a new server with default settings
    craft run myserver
    
    # Stop the server and store a backup
    craft stop myserver
    
    # Start the server again from the latest backup
    craft start myserver
    
    # Create a new backup without interrupting gameplay
    craft backup myserver
    
    # View live server log output
    craft logs myserver
    
    # List running servers
    craft list
    
    # List running and stopped servers
    craft list -a
    
    # Run normal server commands
    craft cmd myserver time set 0600

#### Linux automated backup
This shell script (backup.sh) will save the servers `myserver1` and `myserver2` and log to `~/backup.log`.
Log rotation is built in and `--trim 3` keeps only the 3 most recent backups, removing all others.

    #!/usr/bin/env bash
    ~/go/bin/craft backup myserver1 myserver2 --skip-trim-file-removal-check --trim 3 --log ~/backup.log --log-level info

The following cron job runs it once per hour.

    0 * * * * ~/backup.sh