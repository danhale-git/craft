# Craft
Craft is a simple tool for running and managing Bedrock servers.

Craft is a Docker API wrapper which runs a specific (Bedrock server) container and wraps useful Docker functions.

Windows and Linux are supported, MacOS is currently untested.

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