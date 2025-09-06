package emsub

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	config "github.com/glkeru/EM_Subscriptions/internal/config"
	model "github.com/glkeru/EM_Subscriptions/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sq "github.com/Masterminds/squirrel"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(c *config.Config) (*Repository, error) {
	dsn := "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=" + c.DBSLL

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	return &Repository{pool}, err
}

// создание подписки
func (r *Repository) SubscriptionCreate(ctx context.Context, s model.Subscription) (uuid.UUID, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer conn.Release()

	s.Id = uuid.New()

	sql, arg, err := sq.Insert("subscriptions").
		Columns("id", "service_name", "user_id", "price", "start_date", "end_date").
		Values(s.Id, s.ServiceName, s.UserId, s.Price, s.StartDate, s.EndDate).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return uuid.Nil, err
	}

	_, err = conn.Exec(ctx, sql, arg...)
	if err != nil {
		return uuid.Nil, err
	}

	return s.Id, nil
}

// чтение подписки
func (r *Repository) SubscriptionRead(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	sub := &model.Subscription{}
	row := conn.QueryRow(ctx, "SELECT id, service_name, user_id, price, start_date, end_date FROM subscriptions WHERE id = $1", id)
	err = row.Scan(&sub.Id, &sub.ServiceName, &sub.UserId, &sub.Price, &sub.StartDate, &sub.EndDate)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("subscription %w", model.ErrNotFound)
		}
		return nil, err
	}
	return sub, nil
}

// обновление подписки (PUT)
func (r *Repository) SubscriptionUpdate(ctx context.Context, s model.Subscription) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	sql, args, err := sq.Update("subscriptions").
		Set("service_name", s.ServiceName).
		Set("user_id", s.UserId).
		Set("price", s.Price).
		Set("start_date", s.StartDate).
		Set("end_date", s.EndDate).
		Where(sq.Eq{"id": s.Id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	cmdTag, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("subscription %w", model.ErrNotFound)
	}
	return nil
}

// обновление подписки (PATCH)
func (r *Repository) SubscriptionPatch(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// собрать массивы столбцов и значений
	len := len(fields)
	cols := make([]string, 0, len)
	args := make([]any, 0, len)
	index := 1
	for k, v := range fields {
		cols = append(cols, fmt.Sprintf("%s=$%d", k, index))
		args = append(args, v)
		index++
	}
	args = append(args, id)
	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id=$%d", strings.Join(cols, ","), index)

	_, err = conn.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}

// удаление подписки
func (r *Repository) SubscriptionDelete(ctx context.Context, id uuid.UUID) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	cmdTag, err := conn.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("subscription %w", model.ErrNotFound)
	}
	return nil
}

// список подписок
func (r *Repository) SubscriptionList(ctx context.Context, user uuid.UUID, service_name string, start *time.Time, end *time.Time, limit int, offset int) ([]model.Subscription, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	sqlist := sq.Select("id", "service_name", "user_id", "price", "start_date", "end_date").
		From("subscriptions").
		PlaceholderFormat(sq.Dollar).
		OrderBy("service_name ASC")

	// фильтр: пользователь
	if user != uuid.Nil {
		sqlist = sqlist.Where(sq.Eq{"user_id": user})
	}
	// фильтр: подписка
	if service_name != "" {
		sqlist = sqlist.Where(sq.Eq{"service_name": service_name})
	}
	// фильтр: период
	if start != nil && end != nil {
		sqlist = sqlist.Where(sq.LtOrEq{"start_date": end}).
			Where(sq.Or{
				sq.GtOrEq{"end_date": start},
				sq.Eq{"end_date": nil}})
	} else if start != nil {
		sqlist = sqlist.Where(sq.Or{
			sq.GtOrEq{"end_date": start},
			sq.Eq{"end_date": nil}})
	} else if end != nil {
		sqlist = sqlist.Where(sq.LtOrEq{"start_date": end})
	}

	if limit != 0 {
		sqlist = sqlist.Limit(uint64(limit))
	}
	sqlist = sqlist.Offset(uint64(offset))

	sql, args, err := sqlist.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs := make([]model.Subscription, 0, limit)

	for rows.Next() {
		sub := model.Subscription{}
		err := rows.Scan(&sub.Id, &sub.ServiceName, &sub.UserId, &sub.Price, &sub.StartDate, &sub.EndDate)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// стоимость подписок
func (r *Repository) SubscriptionTotal(ctx context.Context, user uuid.UUID, service_name string, start *time.Time, end *time.Time) (uint, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	// TODO: заморочка с подсчетом месяцев

	return 0, nil
}
