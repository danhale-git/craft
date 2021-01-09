package craft

import (
	"archive/zip"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/danhale-git/craft/internal/files"
)

const (
	worldDirectory = "/bedrock/worlds/Bedrock level" // Default world directory on the server
	mcDirectory    = "/bedrock"                      // Directory with the contents of the mc server zip
	backupDirName  = "craft_backups"                 // Name of the local directory where backups are stored

	saveQueryRetries = 100 // The number of times save query can run without the expected response
	saveQueryDelayMS = 100 // The delay between save query retries, in milliseconds

)

/*// SaveBackup takes a backup from the server and saves it to disk. It returns a pointer to the backup data and the path
// it was saved to.
func SaveBackup(d *DockerClient) (*ServerFiles, string, error) {
	sf := ServerFiles{Docker: d, Archive: &files.Archive{}}
	if err := sf.takeBackup(); err != nil {
		return nil, "", fmt.Errorf("taking server backup")
	}

	path, err := sf.save()
	if err != nil {
		return nil, "", fmt.Errorf("saving backup: %s", err)
	}

	return &sf, path, nil
}*/

/*// RestoreLatestBackup loads backup files from disk and copies them to the DockerClient container.
func RestoreLatestBackup(d *DockerClient) error {
	s := ServerFiles{Docker: d, Archive: &files.Archive{}}

	latestBackup, _, err := LatestServerBackup(d.ContainerName)
	if err != nil {
		return fmt.Errorf("getting most recent backup name: %s", err)
	}

	if err := s.copyFromLocalDisk(path.Join(backupDirectory(), d.ContainerName, latestBackup)); err != nil {
		return fmt.Errorf("taking server backup: %s", err)
	}

	if err := s.restoreBackup(); err != nil {
		return fmt.Errorf("restoring backup files: %s", err)
	}

	return nil
}*/

// ServerFiles copies files to and from the local drive and the server's file system. It requires a DockerClient
// associated with a valid container.
type ServerFiles struct {
	Docker *DockerClient
	*files.Archive
}

// UpdateServerProperties changes the value of a server properties field.
func (s *ServerFiles) UpdateServerProperties(field, value string) error {
	if s.Archive == nil {
		s.Archive = &files.Archive{}
	}

	// Create server properties if it doesn't exist
	if !func() bool {
		for _, f := range s.Archive.Files {
			if f.Name == serverPropertiesFileName {
				return true
			}
		}
		return false
	}() {
		s.AddFile(&files.File{
			Name: serverPropertiesFileName,
			Body: []byte(serverPropertiesDefaultBody),
		})
	}

	// Update the property
	for _, f := range s.Archive.Files {
		if f.Name == serverPropertiesFileName {
			updated, err := setProperty(f.Body, field, value)
			if err != nil {
				return fmt.Errorf("updating file data: %s", err)
			}

			f.Body = updated

			return nil
		}
	}

	panic("no server.properties file exists")
}

func setProperty(data []byte, key, value string) ([]byte, error) {
	lines := strings.Split(string(data), "\n")
	alteredLines := make([]byte, 0)

	changed := false

	// Read file data line by line and amend the chosen key's value
	for _, line := range lines {
		l := strings.TrimSpace(line)

		property := strings.Split(l, "=")

		// Empty line, comment or other property
		if len(l) == 0 || string(l[0]) == "#" || property[0] != key {
			alteredLines = append(alteredLines, []byte(fmt.Sprintf("%s\n", line))...)
			continue
		}

		// Found property, alter value
		alteredLines = append(alteredLines, []byte(fmt.Sprintf("%s=%s\n", key, value))...)
		changed = true
	}

	if !changed {
		return nil, fmt.Errorf("no key was found with name '%s'", key)
	}

	return alteredLines, nil
}

// save writes the backup zip to the default local backup directory. Returns the path the file was saved to or an
// error.
func (s *ServerFiles) save() (string, error) {
	err := s.SaveZip(path.Join(backupDirectory(), s.Docker.ContainerName), s.newBackupFileName())
	if err != nil {
		return "", fmt.Errorf("saving server backup: %s", err)
	}

	return path.Join(backupDirectory(), s.Docker.ContainerName, s.newBackupFileName()), nil
}

// Restore copies the backup files to the server.
func (s *ServerFiles) Restore() error {
	return s.restoreBackup()
}

func (s *ServerFiles) LoadZippedFiles(localPath string) error {
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

	zb, err := ioutil.ReadAll(zf)
	if err != nil {
		return fmt.Errorf("reading file")
	}

	z, err := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
	if err != nil {
		return err
	}

	return s.Archive.LoadZippedFiles(z)
}

// LoadFile adds a file to the backup archive.
func (s *ServerFiles) LoadFile(localPath string) error {
	if s.Docker == nil {
		return errors.New("ServerFiles.Docker may not be nil")
	}

	if s.Archive == nil {
		s.Archive = &files.Archive{}
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file: %s", err)
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("reading file '%s': %s", f.Name(), err)
	}

	s.AddFile(&files.File{
		Name: filepath.Base(localPath),
		Body: b,
	})

	if err = f.Close(); err != nil {
		return fmt.Errorf("closing file: %s", err)
	}

	return nil
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
	for i := 0; i < saveQueryRetries; i++ {
		time.Sleep(saveQueryDelayMS * time.Millisecond)

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
	for _, file := range s.Files {
		if strings.HasSuffix(file.Name, ".mcworld") {
			// Read the file body into another zip archive (double zipped)
			z, err := zip.NewReader(bytes.NewReader(file.Body), int64(len(file.Body)))
			if err != nil {
				return err
			}

			w, err := files.NewArchiveFromZip(z)
			if err != nil {
				return err
			}

			if err = s.Docker.copyTo(worldDirectory, w); err != nil {
				return err
			}
		} else {
			// Other files are copied to the directory containing the mc server executable
			a := files.Archive{}

			a.AddFile(&files.File{
				Name: file.Name,
				Body: file.Body,
			})

			if err := s.Docker.copyTo(mcDirectory, &a); err != nil {
				return err
			}
		}
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

func (s *ServerFiles) newBackupFileName() string {
	return fmt.Sprintf("%s.zip", s.newBackupTimeStamp())
}

func (s *ServerFiles) newBackupTimeStamp() string {
	return fmt.Sprintf("%s_%s",
		s.Docker.ContainerName,
		time.Now().Format(backupFilenameTimeLayout),
	)
}
