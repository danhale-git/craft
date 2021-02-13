package configure

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestSetProperties(t *testing.T) {
	serverProperties := []byte(`changeone=val
nochange=val
changetwo=val
# comment


# commentedfield=val
changethree=val`)

	keys := []string{
		"changeone",
		"changetwo",
		"changethree",
	}

	vals := []string{
		"setone",
		"settwo",
		"setthree",
	}

	res, err := SetProperties(keys, vals, serverProperties)
	if err != nil {
		t.Fatalf("error returned for valid input: %s", err)
	}

	s := bufio.NewScanner(bytes.NewReader(res))
	s.Split(bufio.ScanLines)

	for s.Scan() {
		if strings.HasPrefix(s.Text(), "change") {
			kv := strings.Split(s.Text(), "=")
			if len(kv) != 2 {
				t.Errorf("expected 2 items after splitting '%s' by '=': got %d", s.Text(), len(kv))
			}

			suffix := strings.Replace(kv[0], "change", "", 1)

			want := fmt.Sprintf("set%s", suffix)

			if kv[1] != want {
				t.Errorf("unexpected value in line '%s': expected '%s'", s.Text(), want)
			}
		}
	}
}
