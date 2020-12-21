package files

import (
	"os"
	"path"
)

func save(dirPath, fileName string, body []byte) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(path.Join(dirPath, fileName))
	if err != nil {
		return err
	}

	_, err = f.Write(body)
	if err != nil {
		return err
	}

	return nil
}
