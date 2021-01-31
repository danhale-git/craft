package server

import (
	"path"
)

type FileDetails struct {
	ServerProperties string
	Worlds           string
}

// FileNames are the names of files used by the server.
var FileNames = FileDetails{
	ServerProperties: "server.properties", // File defining the server settings
	Worlds:           "worlds",            // Directory where worlds are stored
}

// FilePaths are the full paths to the files in FileNames.
var FilePaths = FileDetails{
	ServerProperties: path.Join(RootDirectory, "server.properties"), // File defining the server settings
	Worlds:           path.Join(RootDirectory, "worlds"),            // Directory where worlds are stored
}

const (
	RootDirectory = "/bedrock" // Directory where the server files are stored
)
