package process

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
	UUID         string
	ExecutorID   string
	ProcessID    string
	ProcessState string
	Data         string
	State        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type models []model

func convert(c *entity.StoreProcess) (model, error) {
	return model{
		UUID:         c.UUID,
		ExecutorID:   c.ExecutorID,
		ProcessID:    c.ProcessID,
		ProcessState: c.ProcessState,
		Data:         c.Data,
		State:        c.State,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		DeletedAt:    c.DeletedAt,
	}, nil
}

func (m model) convert() (*entity.StoreProcess, error) {
	return &entity.StoreProcess{
		UUID:         m.UUID,
		ExecutorID:   m.ExecutorID,
		ProcessID:    m.ProcessID,
		ProcessState: m.ProcessState,
		Data:         m.Data,
		State:        m.State,

		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}, nil
}

func NewRepository(db dao) *Repository {
	return &Repository{db: db}
}

const (
	ProcessesTableName  = "processes"
	ErrProcessExists    = "process exists"
	ErrProcessNotExists = "process not exists"
	ErrResultQueryEmpty = "result query empty"
	uniqueViolation     = "23505"
)

func (r *Repository) Create(ctx context.Context, c *entity.StoreProcess) error {
	m, err := convert(c)
	if err != nil {
		return err
	}
	m.CreatedAt = time.Now().UTC().Round(time.Millisecond)
	m.UpdatedAt = m.CreatedAt
	err = r.db.DB().WithContext(ctx).
		Table(ProcessesTableName).
		Create(m).Error
	if err != nil {
		if strings.Contains(err.Error(), uniqueViolation) {
			return errors.New(ErrProcessExists)
		}
		return err
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, c *entity.StoreProcess) error {
	m, err := convert(c)
	if err != nil {
		return err
	}
	m.UpdatedAt = time.Now().UTC().Round(time.Millisecond)
	err = r.db.DB().WithContext(ctx).
		Table(ProcessesTableName).
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

func (r *Repository) selectOne(ctx context.Context, db *gorm.DB) (*entity.StoreProcess, error) {
	var (
		m   model
		err error
	)

	tx := db.WithContext(ctx).Table(ProcessesTableName + " AS c")

	err = tx.Take(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(ErrProcessNotExists)
		}
		return nil, err
	}

	return m.convert()
}

func (r *Repository) selectMany(ctx context.Context, db *gorm.DB) ([]*entity.StoreProcess, error) {
	var (
		m   models
		err error
	)

	tx := db.WithContext(ctx).Table(ProcessesTableName + " AS c")

	err = tx.Find(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(ErrProcessNotExists)
		}
		return nil, err
	}
	result := []*entity.StoreProcess{}
	for i := range m {
		c, err := m[i].convert()
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}

	return result, nil
}

func (r *Repository) GetByUUID(ctx context.Context, uuid string) (*entity.StoreProcess, error) {
	return r.selectOne(ctx, r.db.DB().
		Where("c.uuid = ?", uuid))
}

func (r *Repository) Get(ctx context.Context) ([]*entity.StoreProcess, error) {
	return r.selectMany(ctx, r.db.DB().
		Where("c.deleted_at IS NULL"))
}

func (r *Repository) GetByProcessID(ctx context.Context, processID string) ([]*entity.StoreProcess, error) {
	return r.selectMany(ctx, r.db.DB().
		Where("c.process_id = ?", processID).
		Where("c.deleted_at IS NULL"))
}

func (r *Repository) GetNotFinishedByExecutorID(ctx context.Context, executorID string) ([]*entity.StoreProcess, error) {
	return r.selectMany(ctx, r.db.DB().
		Where("(SELECT count(*) FROM processes as c1 WHERE length(c1.process_state) = 0 AND c1.process_id = c.process_id) = 0").
		Where("c.executor_id = ?", executorID).
		//Where("c.process_id = c1.process_id").
		Where("c.deleted_at IS NULL"))
}

func (r *Repository) DeleteByUUID(ctx context.Context, resourceUUID string) error {
	dt := time.Now().UTC().Round(time.Millisecond)
	m := model{
		UUID:      resourceUUID,
		DeletedAt: &dt,
	}
	err := r.db.DB().WithContext(ctx).
		Table(ProcessesTableName).
		Select("*").
		Where("uuid = ?", m.UUID).
		Where("deleted_at IS NULL").
		Omit(
			"uuid",
			"executor_id",
			"process_id",
			"state",
			"process_state",
			"data",
			"created_at",
		).
		Updates(m).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DeleteByProcessID(ctx context.Context, processID string) error {
	dt := time.Now().UTC().Round(time.Millisecond)
	m := model{
		ProcessID: processID,
		DeletedAt: &dt,
	}
	err := r.db.DB().WithContext(ctx).
		Table(ProcessesTableName).
		Select("*").
		Where("process_id = ?", m.ProcessID).
		Where("deleted_at IS NULL").
		Omit(
			"uuid",
			"executor_id",
			"process_id",
			"state",
			"process_state",
			"data",
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
		Table(ProcessesTableName).
		Where("deleted_at IS NOT NULL").
		Delete(m).Error
}

func (r *Repository) GetView(ctx context.Context, pagination *internalEntity.Pagination,
	filter *internalEntity.Filter) ([]*entity.StoreProcess, int, error) {
	tx := r.db.DB().Begin()

	var (
		m   models
		err error
	)

	tx1 := tx.WithContext(ctx).Table(ProcessesTableName + " AS c")

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
			return nil, 0, errors.New(ErrProcessNotExists)
		}
		return nil, 0, err
	}

	processes := []*entity.StoreProcess{}
	for i := range m {
		c, err := m[i].convert()
		if err != nil {
			tx.Rollback()
			return nil, 0, err
		}
		processes = append(processes, c)
	}

	tx2 := tx.WithContext(ctx).Table(ProcessesTableName + " AS c")

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

	return processes, count, nil
}
