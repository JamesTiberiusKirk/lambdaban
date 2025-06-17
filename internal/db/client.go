package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/JamesTiberiusKirk/lambdaban/internal/metrics"
	"github.com/JamesTiberiusKirk/lambdaban/internal/models"
	"github.com/JamesTiberiusKirk/migrator/migrator"
)

type Client struct {
	log     *slog.Logger
	m       *metrics.Metrics
	connUrl string
	db      *sql.DB
	sq      squirrel.StatementBuilderType
	now     func() time.Time
}

// InitClient initializes a new database client and pings the DB.
func InitClient(
	log *slog.Logger,
	m *metrics.Metrics,
	user, pass, host, dbName string,
	disableSSL bool,
	now func() time.Time,
) (*Client, error) {
	connUrl := fmt.Sprintf("postgres://%s:%s@%s/%s", user, pass, host, dbName)
	if disableSSL {
		connUrl += "?sslmode=disable"
	}

	db, err := sql.Open("postgres", connUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	migrate, err := migrator.NewMigratorWithSqlClient(db, "./internal/db/sql/")
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator instance: %w", err)
	}

	err = migrate.ApplySchemaUp()
	if err != nil && !errors.Is(err, migrator.ErrSchemaAlreadyInitialised) {
		return nil, fmt.Errorf("failed to apply schema up: %w", err)
	}

	err = migrate.ApplyMigration()
	if err != nil {
		return nil, fmt.Errorf("failed to apply schema up: %w", err)
	}

	return &Client{
		log:     log,
		m:       m,
		connUrl: connUrl,
		db:      db,
		sq:      squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		now:     now,
	}, nil
}

// InitTTLCleanup starts a background goroutine that deletes users whose updated_at is older than olderThan,
// running at the given interval. It stops when the provided context is cancelled.
func (c *Client) InitTTLCleanup(ctx context.Context, interval, olderThan time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Use Squirrel to build the DELETE statement
				cutoff := c.now().Add(-olderThan)
				sqlStr, args, err := c.sq.
					Delete("users").
					Where(squirrel.Lt{"updated_at": cutoff}).
					ToSql()
				if err != nil {
					c.log.Error("TTL cleanup SQL build failed", "error", err)
					continue
				}
				_, err = c.db.ExecContext(ctx, sqlStr, args...)
				if err != nil {
					c.log.Error("TTL cleanup failed", "error", err)
				} else {
					c.log.Info("TTL cleanup ran successfully")
				}
			case <-ctx.Done():
				c.log.Info("TTL cleanup worker stopped")
				return
			}
		}
	}()
}

// AddToUser adds a ticket to the user's tickets array in the users table.
func (c *Client) AddToUser(ctx context.Context, id string, ticket models.Ticket) error {
	// Fetch existing tickets
	var ticketsJSON []byte
	sqlStr, args, err := c.sq.
		Select("tickets").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	row := c.db.QueryRowContext(ctx, sqlStr, args...)
	if err := row.Scan(&ticketsJSON); err != nil {
		return err
	}

	var tickets []models.Ticket
	if err := json.Unmarshal(ticketsJSON, &tickets); err != nil {
		return err
	}

	// Append new ticket
	tickets = append(tickets, ticket)
	newTicketsJSON, err := json.Marshal(tickets)
	if err != nil {
		return err
	}
	updateSQL, updateArgs, err := c.sq.
		Update("users").
		Set("tickets", newTicketsJSON).
		Set("updated_at", c.now()).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, updateSQL, updateArgs...)
	return err
}

// CreateUser creates a new user and returns the user's ID.
func (c *Client) CreateUser(ctx context.Context) (string, error) {
	defaultTicketsString, err := json.Marshal(defaultTickets)
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	insertSQL, insertArgs, err := c.sq.
		Insert("users").
		Columns("id", "tickets", "updated_at").
		Values(id, []byte(defaultTicketsString), c.now()).
		ToSql()
	if err != nil {
		return "", err
	}
	_, err = c.db.ExecContext(ctx, insertSQL, insertArgs...)
	if err != nil {
		return "", err
	}
	return id, nil
}

// DeleteTodoByUserAndTodoId removes a specific ticket from a user's tickets array.
func (c *Client) DeleteTodoByUserAndTodoId(ctx context.Context, userId string, todoId string) error {
	// Fetch tickets
	var ticketsJSON []byte
	sqlStr, args, err := c.sq.
		Select("tickets").
		From("users").
		Where(squirrel.Eq{"id": userId}).
		ToSql()
	if err != nil {
		return err
	}
	row := c.db.QueryRowContext(ctx, sqlStr, args...)
	if err := row.Scan(&ticketsJSON); err != nil {
		return err
	}
	var tickets []models.Ticket
	if err := json.Unmarshal(ticketsJSON, &tickets); err != nil {
		return err
	}
	// Remove ticket by todoId
	newTickets := make([]models.Ticket, 0, len(tickets))
	for _, t := range tickets {
		if t.Id != todoId {
			newTickets = append(newTickets, t)
		}
	}
	newTicketsJSON, err := json.Marshal(newTickets)
	if err != nil {
		return err
	}
	updateSQL, updateArgs, err := c.sq.
		Update("users").
		Set("tickets", newTicketsJSON).
		Set("updated_at", c.now()).
		Where(squirrel.Eq{"id": userId}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, updateSQL, updateArgs...)
	return err
}

// DeleteUserByID deletes a user by ID.
func (c *Client) DeleteUserByID(ctx context.Context, id string) error {
	delSQL, delArgs, err := c.sq.
		Delete("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, delSQL, delArgs...)
	return err
}

// GetAllByUser returns all tickets for a user.
func (c *Client) GetAllByUser(ctx context.Context, id string) ([]models.Ticket, error) {
	sqlStr, args, err := c.sq.
		Select("tickets").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}
	row := c.db.QueryRowContext(ctx, sqlStr, args...)
	var ticketsJSON []byte
	if err := row.Scan(&ticketsJSON); err != nil {
		return nil, err
	}
	var tickets []models.Ticket
	if err := json.Unmarshal(ticketsJSON, &tickets); err != nil {
		return nil, err
	}
	return tickets, nil
}

// GetAllByUserSplitByStatus returns tickets split by status.
func (c *Client) GetAllByUserSplitByStatus(ctx context.Context, id string) (todo, inProgress, done []models.Ticket, err error) {
	tickets, err := c.GetAllByUser(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, t := range tickets {
		switch t.Status {
		case models.StatusTodo:
			todo = append(todo, t)
		case models.StatusInProgress:
			inProgress = append(inProgress, t)
		case models.StatusDone:
			done = append(done, t)
		}
	}
	return todo, inProgress, done, nil
}

// UpdateUser replaces the user's tickets array.
func (c *Client) UpdateUser(ctx context.Context, userId string, tickets []models.Ticket) error {
	ticketsJSON, err := json.Marshal(tickets)
	if err != nil {
		return err
	}
	updateSQL, updateArgs, err := c.sq.
		Update("users").
		Set("tickets", ticketsJSON).
		Set("updated_at", c.now()).
		Where(squirrel.Eq{"id": userId}).
		ToSql()
	if err != nil {
		return err
	}
	_, err = c.db.ExecContext(ctx, updateSQL, updateArgs...)
	return err
}
