package frontuser

import (
	"context"
	"time"

	"github.com/wanderer69/flow_processor/pkg/entity"
	"gorm.io/gorm"
)

type dao interface {
	DB() *gorm.DB
}

type Repository struct {
	db dao
}

type model struct {
	UUID      string
	Login     string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type models []model

func convert(c *entity.FrontUser) (model, error) {
	return model{
		UUID:  c.UUID,
		Login: c.Login,
		//		CreatedAt:    c.CreatedAt,
		//		UpdatedAt:    c.UpdatedAt,
		//		DeletedAt:    c.DeletedAt,
	}, nil
}

func (m model) convert() (*entity.FrontUser, error) {
	return &entity.FrontUser{
		UUID:  m.UUID,
		Login: m.Login,
		//		CreatedAt: m.CreatedAt,
		//		UpdatedAt: m.UpdatedAt,
		//		DeletedAt: m.DeletedAt,
	}, nil
}

func NewRepository(db dao) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Register(ctx context.Context, login, password string) error {
	return nil
}

func (r *Repository) Get(ctx context.Context, login, password string) error {
	return nil
}
