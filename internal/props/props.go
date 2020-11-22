package props

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// SetProperty reads a server.properties file line by line and amends the value of the given key wit the given value
func SetProperty(filePath, key, value string) error {
	// Read server.properties file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file at %s: %s", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	alteredLines := make([]string, 0)

	// Append lines to alteredLines, altering value if found
	valueChanged := false

	for _, line := range lines {
		l := strings.TrimSpace(line)

		property := strings.Split(l, "=")

		// Empty line, comment or other property
		if len(l) == 0 || string(l[0]) == "#" || property[0] != key {
			alteredLines = append(alteredLines, line, "\n")
			continue
		}

		// Found property, alter value
		alteredLines = append(alteredLines, fmt.Sprintf("%s=%s\n", key, value))

		valueChanged = true

		log.Printf("set '%s' to '%s'", key, value)
	}

	if valueChanged {
		err = writeLines(filePath, alteredLines)
		if err != nil {
			return fmt.Errorf("writing to file at %s: %s", filePath, err)
		}
	} else {
		return fmt.Errorf("no flag found with key '%s'", key)
	}

	return nil
}

func writeLines(file string, lines []string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	defer w.Flush()

	for _, line := range lines {
		_, err := w.WriteString(line)
		if err != nil {
			return err
		}
	}

	return nil
}
