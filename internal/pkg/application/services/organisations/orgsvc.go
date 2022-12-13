package organisations

import (
	"errors"
	"io"

	"github.com/diwise/api-opendata/internal/pkg/domain"
)

type Registry interface {
	Get(organisationID string) (*domain.Organisation, error)
}

func NewRegistry(input io.Reader) (Registry, error) {
	return &registry{}, nil
}

type registry struct {
}

func (r *registry) Get(organisationID string) (*domain.Organisation, error) {
	return nil, errors.New("not implemented")
}
