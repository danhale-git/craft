package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/danhale-git/craft/internal/mock"
)

func TestServer_Command(t *testing.T) {
	mockClient := &mock.ContainerAPIDockerClientMock{}
	s := &Server{ContainerAPIClient: mockClient}
	conn, reader := net.Pipe()
	mockClient.Conn = conn
	mockClient.Reader = bufio.NewReader(reader)

	c := []string{"arg1", "arg2", "arg3"}

	go func() {
		err := s.Command(c)
		if err != nil {
			t.Errorf("error returned for valid input")
		}
	}()

	want := strings.Join(c, " ") + "\n"

	got, err := mockClient.Reader.ReadString('\n')
	if err != nil {
		t.Errorf("error reading command input from mock client: %s", err)
	}

	if got != want {
		t.Errorf("want: %s got: %s", want, got)
	}
}

func TestServer_LogReader(t *testing.T) {
	s := &Server{ContainerAPIClient: &mock.ContainerAPIDockerClientMock{}}

	r, err := s.LogReader(20)
	if err != nil {
		t.Errorf("error returned for valid input: %s", err)
	}

	if r == nil {
		t.Errorf("reader is nil")
	}

	_, err = r.ReadString('!')
	if err != nil {
		t.Errorf("error reading from valid reader: %s", err)
	}
}

func TestContainerID(t *testing.T) {
	s := &Server{ContainerAPIClient: &mock.ContainerAPIDockerClientMock{}}

	for i := 1; i <= 3; i++ {
		want := fmt.Sprintf("mc%d_ID", i)
		got, err := containerID(fmt.Sprintf("mc%d", i), s)

		if err != nil {
			t.Errorf("error returned for valid input: %s", err)
		}

		if got != want {
			t.Errorf("want: %s got: %s", want, got)
		}
	}
}
