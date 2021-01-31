package configure

import (
	"fmt"
	"strings"
)

// SetProperties applies the given set of values to their corresponding keys from two parallel slices. The data should
// be the contents of a complete (created from default) server.properties file. Missing keys will not be created and
// will throw an error.
func SetProperties(k, v []string, data []byte) ([]byte, error) {
	if len(k) != len(v) {
		return nil, fmt.Errorf("key and value collections were different lengths, should be one value per key")
	}

	for i := 0; i < len(k); i++ {
		var err error
		data, err = setProperty(data, k[i], v[i])

		if err != nil {
			return nil, err
		}
	}

	return data, nil
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
