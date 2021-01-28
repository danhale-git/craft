package configure

import (
	"fmt"
	"strings"
)

/*if f.Name == docker.serverPropertiesFileName {
	updated, err := setProperty(f.Body, field, value)
	if err != nil {
		return fmt.Errorf("updating file data: %s", err)
	}

	f.Body = updated

	return nil
}*/
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
