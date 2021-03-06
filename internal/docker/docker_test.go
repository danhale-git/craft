package docker

import (
	"fmt"
	"testing"
)

func TestContainerID(t *testing.T) {
	d := &Container{ContainerAPIClient: &ContainerListDockerClientMock{}}

	for i := 1; i <= 3; i++ {
		want := fmt.Sprintf("mc%d_ID", i)
		got, err := ContainerID(fmt.Sprintf("mc%d", i), d)

		if err != nil {
			t.Errorf("error returned for valid input: %s", err)
		}

		if got != want {
			t.Errorf("want: %s got: %s", want, got)
		}
	}
}
