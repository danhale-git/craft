package server

import (
	"path"
)

const (
	Directory = "/bedrock" // Directory where the server files are stored
)

type FileDetails struct {
	ServerProperties string
	Worlds           string
	DefaultWorld     string
}

// FileNames are the names of files used by the server.
var FileNames = FileDetails{ //nolint:gochecknoglobals
	ServerProperties: "server.properties", // File defining the server settings
	Worlds:           "worlds",            // Directory where worlds are stored
	DefaultWorld:     "Bedrock level",     // Directory where the default world is stored
}

// LocalPaths are the paths to server files from the server directory (server.Directory).
var LocalPaths = FileDetails{ //nolint:gochecknoglobals
	ServerProperties: path.Join(FileNames.ServerProperties),               // File defining the server settings
	Worlds:           path.Join(FileNames.Worlds),                         // Directory where worlds are stored
	DefaultWorld:     path.Join(FileNames.Worlds, FileNames.DefaultWorld), // Directory where the default world is stored
}

// FullPaths are the full paths to server files, from the root directory.
var FullPaths = FileDetails{ //nolint:gochecknoglobals
	ServerProperties: path.Join(Directory, FileNames.ServerProperties),               // File defining the server settings
	Worlds:           path.Join(Directory, FileNames.Worlds),                         // Directory where worlds are stored
	DefaultWorld:     path.Join(Directory, FileNames.Worlds, FileNames.DefaultWorld), // Directory where the default world is stored
}
