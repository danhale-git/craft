![Go Report Card](https://goreportcard.com/badge/github.com/danhale-git/craft)
![coverage](https://img.shields.io/badge/coverage-43.1%25-orange)
![coverage](https://img.shields.io/badge/build-passing-brightgreen)
![example workflow](https://github.com/danhale-git/craft/actions/workflows/go-test.yaml/badge.svg)
![example workflow](https://github.com/danhale-git/craft/actions/workflows/test-coverage.yaml/badge.svg)

# Craft
Craft is a simple tool for running and managing Bedrock servers.

It's a docker API wrapper which runs a specific container and provides useful functions such as backing up the server.

Windows and Linux are supported.

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