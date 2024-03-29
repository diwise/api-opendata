package organisations

import (
	"bytes"
	"testing"

	"github.com/matryer/is"
)

func TestLoad(t *testing.T) {
	is := is.New(t)

	config := bytes.NewBufferString(configFile)
	svc, err := NewRegistry(config)
	is.NoErr(err)

	org, err := svc.Get("test0")

	is.NoErr(err)
	is.Equal(org.Name, "foo")
}

func TestLoadNoInfile(t *testing.T) {
	is := is.New(t)

	svc, err := NewRegistry(nil)
	is.NoErr(err)

	_, err = svc.Get("test0")

	is.True(err != nil)
	is.Equal(err.Error(), "organisation test0 not found")
}

const configFile string = `
organisations:
  - id: test0
    name: foo
  - id: test1
    name: bar
`
