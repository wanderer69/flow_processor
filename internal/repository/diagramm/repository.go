package diagramm

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	internalEntity "github.com/wanderer69/flow_processor/internal/entity"
	"github.com/wanderer69/flow_processor/pkg/entity"
)

type dao interface {
	DB() *gorm.DB
}

type Repository struct {
	db dao
}

type model struct {
	UUID      string
	Data      string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type models []model

func convert(c *entity.Diagramm) (model, error) {
	return model{
		UUID:      c.UUID,
		Data:      c.Data,
		Name:      c.Name,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		DeletedAt: c.DeletedAt,
	}, nil
}

func (m model) convert() (*entity.Diagramm, error) {
	return &entity.Diagramm{
		UUID:      m.UUID,
		Data:      m.Data,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}, nil
}

func NewRepository(db dao) *Repository {
	return &Repository{db: db}
}

const (
	DiagrammesTableName  = "diagramms"
	ErrDiagrammExists    = "diagramm exists"
	ErrDiagrammNotExists = "diagramm not exists"
	ErrResultQueryEmpty  = "result query empty"
	uniqueViolation      = "23505"
)

func (r *Repository) Create(ctx context.Context, c *entity.Diagramm) error {
	m, err := convert(c)
	if err != nil {
		return err
	}
	m.CreatedAt = time.Now().UTC().Round(time.Millisecond)
	m.UpdatedAt = m.CreatedAt
	err = r.db.DB().WithContext(ctx).
		Table(DiagrammesTableName).
		Create(m).Error
	if err != nil {
		if strings.Contains(err.Error(), uniqueViolation) {
			return errors.New(ErrDiagrammExists)
		}
		return err
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, c *entity.Diagramm) error {
	m, err := convert(c)
	if err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC().Round(time.Millisecond)
	err = r.db.DB().WithContext(ctx).
		Table(DiagrammesTableName).
		Select("*").
		Where("uuid = ?", m.UUID).
		Omit(
			"uuid",
			"created_at",
		).
		Updates(m).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) selectOne(ctx context.Context, db *gorm.DB) (*entity.Diagramm, error) {
	var (
		m   model
		err error
	)

	tx := db.WithContext(ctx).Table(DiagrammesTableName + " AS c")

	err = tx.Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(ErrDiagrammNotExists)
		}
		return nil, err
	}

	return m.convert()
}

func (r *Repository) selectMany(ctx context.Context, db *gorm.DB) ([]*entity.Diagramm, error) {
	var (
		m   models
		err error
	)

	tx := db.WithContext(ctx).Table(DiagrammesTableName + " AS c")

	err = tx.Find(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(ErrDiagrammNotExists)
		}
		return nil, err
	}
	result := []*entity.Diagramm{}
	for i := range m {
		c, err := m[i].convert()
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}

	return result, nil
}

func (r *Repository) GetByUUID(ctx context.Context, uuid string) (*entity.Diagramm, error) {
	return r.selectOne(ctx, r.db.DB().
		Where("c.uuid = ?", uuid))
}

func (r *Repository) Get(ctx context.Context) ([]*entity.Diagramm, error) {
	return r.selectMany(ctx, r.db.DB().
		Where("c.deleted_at IS NULL"))
}

func (r *Repository) GetByName(ctx context.Context, name string) ([]*entity.Diagramm, error) {
	return r.selectMany(ctx, r.db.DB().
		Where("c.name = ?", name).
		Where("c.deleted_at IS NULL"))
}

func (r *Repository) DeleteByUUID(ctx context.Context, resourceUUID string) error {
	dt := time.Now().UTC().Round(time.Millisecond)
	m := model{
		UUID:      resourceUUID,
		DeletedAt: &dt,
	}
	err := r.db.DB().WithContext(ctx).
		Table(DiagrammesTableName).
		Select("*").
		Where("uuid = ?", m.UUID).
		Where("deleted_at IS NULL").
		Omit(
			"uuid",
			"data",
			"name",
			"created_at",
		).
		Updates(m).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteByName(ctx context.Context, name string) error {
	dt := time.Now().UTC().Round(time.Millisecond)
	m := model{
		Name:      name,
		DeletedAt: &dt,
	}
	err := r.db.DB().WithContext(ctx).
		Table(DiagrammesTableName).
		Select("*").
		Where("name = ?", m.UUID).
		Where("deleted_at IS NULL").
		Omit(
			"uuid",
			"data",
			"name",
			"created_at",
		).
		Updates(m).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteByDeletedAt(ctx context.Context) error {
	m := model{}
	return r.db.DB().WithContext(ctx).
		Table(DiagrammesTableName).
		Where("deleted_at IS NOT NULL").
		Delete(m).Error
}

func (r *Repository) GetView(ctx context.Context, pagination *internalEntity.Pagination,
	filter *internalEntity.Filter) ([]*entity.Diagramm, int, error) {
	tx := r.db.DB().Begin()

	var (
		m   models
		err error
	)

	tx1 := tx.WithContext(ctx).Table(DiagrammesTableName + " AS c")

	if pagination != nil {
		tx1 = tx1.Offset(pagination.Count)
		if pagination.Size > 0 {
			tx1 = tx1.Limit(pagination.Size)
		}
	}

	if len(filter.SortByFieldName) > 0 {
		key := filter.SortByFieldName
		switch filter.SortValue {
		case "asc":
			tx1 = tx1.Order(key)
		case "desc":
			tx1 = tx1.Order(key + " DESC")
		}
	}

	for i := range filter.Fields {
		switch filter.Fields[i].Type {
		case "=":
			condition := filter.Fields[i].FieldName + " = ?"
			tx1 = tx1.Where(condition, filter.Fields[i].Value)
		case "BETWEEN":
			condition := filter.Fields[i].FieldName + " BETWEEN ? AND ?"
			tx1 = tx1.Where(condition, filter.Fields[i].Value, filter.Fields[i].ValueUp)
		case "IN":
			lst := []string{}
			for j := range filter.Fields[i].Set {
				lst = append(lst, filter.Fields[i].Set[j].Value)
			}
			condition := filter.Fields[i].FieldName + " IN ?"
			tx1 = tx1.Where(condition, lst)
		case "ISNULL":
			condition := filter.Fields[i].FieldName + " IS NULL"
			tx1 = tx1.Where(condition)
		case "!ISNULL":
			condition := filter.Fields[i].FieldName + " IS NOT NULL"
			tx1 = tx1.Where(condition)
		}
	}

	err = tx1.Find(&m).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New(ErrDiagrammNotExists)
		}
		return nil, 0, err
	}

	diagrammes := []*entity.Diagramm{}
	for i := range m {
		c, err := m[i].convert()
		if err != nil {
			tx.Rollback()
			return nil, 0, err
		}
		diagrammes = append(diagrammes, c)
	}

	tx2 := tx.WithContext(ctx).Table(DiagrammesTableName + " AS c")

	tx2 = tx2.Select("COUNT(*)")

	for i := range filter.Fields {
		switch filter.Fields[i].Type {
		case "=":
			condition := filter.Fields[i].FieldName + " = ?"
			tx2 = tx2.Where(condition, filter.Fields[i].Value)
		case "BETWEEN":
			condition := filter.Fields[i].FieldName + " BETWEEN ? AND ?"
			tx2 = tx2.Where(condition, filter.Fields[i].Value, filter.Fields[i].ValueUp)
		case "IN":
			lst := []string{}
			for j := range filter.Fields[i].Set {
				lst = append(lst, filter.Fields[i].Set[j].Value)
			}
			condition := filter.Fields[i].FieldName + " IN ?"
			tx2 = tx2.Where(condition, lst)
		case "ISNULL":
			condition := filter.Fields[i].FieldName + " IS NULL"
			tx2 = tx2.Where(condition)
		case "!ISNULL":
			condition := filter.Fields[i].FieldName + " IS NOT NULL"
			tx2 = tx2.Where(condition)
		}
	}

	count := 0
	err = tx2.Find(&count).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New(ErrResultQueryEmpty)
		}
		return nil, 0, err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	return diagrammes, count, nil
}
