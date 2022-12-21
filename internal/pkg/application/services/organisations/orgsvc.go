package organisations

import (
	"fmt"
	"io"
	"reflect"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"github.com/diwise/api-opendata/internal/pkg/domain"
)

type organisation struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type Registry interface {
	Get(organisationID string) (*domain.Organisation, error)
}

func NewRegistry(input io.Reader) (Registry, error) {

	var err error
	reg := &registry{
		orgs: make(map[string]*domain.Organisation),
	}

	if inputIsNotNil(input) {
		buf, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}

		cfg := &struct {
			Orgs []organisation `yaml:"organisations"`
		}{}

		err = yaml.Unmarshal(buf, &cfg)
		if err != nil {
			return nil, err
		}

		for _, org := range cfg.Orgs {
			reg.orgs[org.ID] = &domain.Organisation{Name: org.Name}
		}
	}

	return reg, err
}

type registry struct {
	orgs map[string]*domain.Organisation
	mu   sync.Mutex
}

func (r *registry) Get(organisationID string) (*domain.Organisation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	org, ok := r.orgs[organisationID]
	if !ok {
		return nil, fmt.Errorf("organisation %s not found", organisationID)
	}
	return org, nil
}

func inputIsNotNil(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return !reflect.ValueOf(v).IsNil()
	}
	return true
}
