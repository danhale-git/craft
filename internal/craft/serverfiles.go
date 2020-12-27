package craft

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/files"

	"github.com/mitchellh/go-homedir"
)

const backupDirName = "craft_backups"

// ServerFiles copies files to and from the local drive and the server's file system. It requires a DockerClient
// associated with a valid container.
type ServerFiles struct {
	Docker *DockerClient
	*files.Archive
}

// NewBackup takes a backup from the server with the given name.
func NewBackup(d *DockerClient) (*ServerFiles, error) {
	sb := ServerFiles{Docker: d, Archive: &files.Archive{}}
	if err := sb.takeBackup(); err != nil {
		return nil, fmt.Errorf("taking server backup")
	}

	return &sb, nil
}

// RestoreLatestBackup loads backup files from disk and copies them to the DockerClient container.
func RestoreLatestBackup(d *DockerClient) error {
	s := ServerFiles{Docker: d, Archive: &files.Archive{}}

	latestBackup, _, err := LatestServerBackup(d.containerName)
	if err != nil {
		return fmt.Errorf("getting most recent backup name: %s", err)
	}

	if err := s.copyFromLocalDisk(path.Join(backupDirectory(), d.containerName, latestBackup)); err != nil {
		return fmt.Errorf("taking server backup: %s", err)
	}

	if err := s.restoreBackup(); err != nil {
		return fmt.Errorf("restoring backup files: %s", err)
	}

	return nil
}

// LoadWorld adds a file to the backup archive.
func (s *ServerFiles) LoadFile(localPath string) error {
	if s.Docker == nil {
		return errors.New("ServerFiles.Docker may not be nil")
	}

	if s.Archive == nil {
		s.Archive = &files.Archive{}
	}

	zf, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %s", err)
	}

	b, err := ioutil.ReadAll(zf)
	if err != nil {
		return fmt.Errorf("reading file '%s': %s", zf.Name(), err)
	}

	s.AddFile(&files.File{
		Name: filepath.Base(localPath),
		Body: b,
	})

	if err = zf.Close(); err != nil {
		return fmt.Errorf("closing file: %s", err)
	}

	return nil
}

// Saves the backup to the default local backup directory. Returns the path the file was saved to or an error.
func (s *ServerFiles) Save() (string, error) {
	err := s.SaveZip(path.Join(backupDirectory(), s.Docker.containerName), s.newBackupFileName())
	if err != nil {
		return "", fmt.Errorf("saving server backup: %s", err)
	}

	return path.Join(backupDirectory(), s.Docker.containerName, s.newBackupFileName()), nil
}

// Restore copies the backup files to the server.
func (s *ServerFiles) Restore() error {
	return s.restoreBackup()
}

// Backup runs the save hold/query/resume command sequence and saves a .mcworld file snapshot to the given local path.
func (s *ServerFiles) takeBackup() error {
	logs, err := s.Docker.LogReader(0)

	// save hold
	saveHold, err := s.commandResponse("save hold", logs)
	if err != nil {
		return err
	}

	switch strings.TrimSpace(saveHold) {
	case "Saving...":
	case "The command is already running":
		break
	default:
		return fmt.Errorf("unexpected response to `save hold`: '%s'", saveHold)
	}

	// Query until ready for backup
	// TODO: throw an error when retries are exceeded
	for i := 0; i < saveHoldQueryRetries; i++ {
		time.Sleep(saveHoldDelayMilliseconds * time.Millisecond)

		saveQuery, err := s.commandResponse("save query", logs)
		if err != nil {
			return err
		}

		// Ready for backup
		if strings.HasPrefix(saveQuery, "Data saved. Files are now ready to be copied.") {
			// A second line is returned with a list of files, read it to discard it.
			if _, err := logs.ReadString('\n'); err != nil {
				return fmt.Errorf("reading 'save query' file list response: %s", err)
			}

			// Copy backup files from server
			err = s.copyFromServer()
			if err != nil {
				return fmt.Errorf("backing up server: %s", err)
			}

			break
		}
	}

	// save resume
	saveResume, err := s.commandResponse("save resume", logs)
	if err != nil {
		return err
	}

	if strings.TrimSpace(saveResume) != "Changes to the level are resumed." {
		return fmt.Errorf("unexpected response to `save resume`: '%s'", saveResume)
	}

	return nil
}

func (s *ServerFiles) commandResponse(cmd string, logs *bufio.Reader) (string, error) {
	err := s.Docker.Command(strings.Split(cmd, " "))
	if err != nil {
		return "", fmt.Errorf("running command `%s`: %s", cmd, err)
	}

	// Read command echo to discard it
	if _, err := logs.ReadString('\n'); err != nil {
		return "", fmt.Errorf("retrieving echo for command `%s`: %s", cmd, err)
	}

	// Read command response
	response, err := logs.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("retrieving `%s` response: %s", cmd, err)
	}

	return response, nil
}

func (s *ServerFiles) restoreBackup() error {
	foundWorld := false

	for _, file := range s.Files {
		if strings.HasSuffix(file.Name, ".mcworld") {
			// World is copied into the the active world directory.
			foundWorld = true

			// Read the file body into another zip archive (double zipped)
			z, err := zip.NewReader(bytes.NewReader(file.Body), int64(len(file.Body)))
			if err != nil {
				return err
			}

			w, err := files.NewArchiveFromZip(z)
			if err != nil {
				return err
			}

			err = s.Docker.copyTo(worldDirectory, w)
			if err != nil {
				return err
			}
		} else {
			// Other files are copied to the directory containing the mc server executable
			a := files.Archive{}

			a.AddFile(&files.File{
				Name: file.Name,
				Body: file.Body,
			})

			return s.Docker.copyTo(mcDirectory, &a)
		}
	}

	if !foundWorld {
		return fmt.Errorf("no .mcworld file present in backup")
	}

	return nil
}

func (s *ServerFiles) copyFromLocalDisk(localPath string) error {
	// Open a zip archive for reading.
	z, err := zip.OpenReader(localPath)
	if err != nil {
		return err
	}

	for _, file := range z.File {
		f, err := file.Open()
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		s.AddFile(&files.File{
			Name: file.Name,
			Body: b,
		})
	}

	return z.Close()
}

func (s *ServerFiles) copyFromServer() error {
	// Back up world
	mcworldFile, err := s.copyWorldFromContainer()
	if err != nil {
		return fmt.Errorf("copying world data from container: %s", err)
	}

	s.AddFile(mcworldFile)

	// Back up settings
	serverPropertiesFile, err := s.copyServerPropertiesFromContainer()
	if err != nil {
		return fmt.Errorf("copying server properties from container")
	}

	s.AddFile(serverPropertiesFile)

	return nil
}

func (s *ServerFiles) copyWorldFromContainer() (*files.File, error) {
	// Copy the world directory and it's contents from the container
	a, err := s.Docker.copyFrom(worldDirectory)
	if err != nil {
		return nil, err
	}

	// Remove 'Bedrock level' directory
	newFiles := make([]*files.File, 0)

	for _, f := range a.Files {
		f.Name = strings.Replace(f.Name, "Bedrock level/", "", 1)

		// Skip the file representing the 'Bedrock level' directory
		if len(strings.TrimSpace(f.Name)) == 0 {
			continue
		}

		newFiles = append(newFiles, f)
	}

	a.Files = newFiles

	wz, err := a.Zip()
	if err != nil {
		return nil, fmt.Errorf("converting world archive to zip: %s", err)
	}

	mcwFile := files.File{
		Name: fmt.Sprintf("%s.mcworld", s.newBackupTimeStamp()),
		Body: wz.Bytes(),
	}

	return &mcwFile, nil
}

func (s *ServerFiles) copyServerPropertiesFromContainer() (*files.File, error) {
	a, err := s.Docker.copyFrom(path.Join(mcDirectory, serverPropertiesFileName))
	if err != nil {
		return nil, fmt.Errorf("copying '%s' from container path %s: %s", serverPropertiesFileName, mcDirectory, err)
	}

	serverProperties := files.File{
		Name: serverPropertiesFileName,
		Body: a.Files[0].Body,
	}

	return &serverProperties, nil
}

func backupDirectory() string {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	return path.Join(home, backupDirName)
}

func (s *ServerFiles) newBackupFileName() string {
	return fmt.Sprintf("%s.zip", s.newBackupTimeStamp())
}

func (s *ServerFiles) newBackupTimeStamp() string {
	return fmt.Sprintf("%s_%s",
		s.Docker.containerName,
		time.Now().Format(backupFilenameTimeLayout),
	)
}

// // // //
