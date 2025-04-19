package db

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/models"
	"github.com/google/uuid"
)

var (
	state   globalState
	stateMu sync.RWMutex
	ttl     = 2 * time.Minute
)

type user struct {
	tickets     []models.Ticket
	lastUpdated time.Time
}

type globalState struct {
	users map[string]user
}

type InMemClient struct {
	log *slog.Logger
}

func NewInMemClient(log *slog.Logger) *InMemClient {
	stateMu.Lock()
	state = globalState{
		users: map[string]user{},
	}
	stateMu.Unlock()
	return &InMemClient{
		log: log,
	}
}

var (
	ErrUserNotFound = fmt.Errorf("user not found")
)

func (db *InMemClient) InitTTLCleanup() {
	db.log.Info("Initialised TTL cleanup", "TTL", ttl)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			<-ticker.C
			now := time.Now()
			stateMu.Lock()
			for id, u := range state.users {
				if now.Sub(u.lastUpdated) > ttl {
					db.log.Info("Cleaned up user", "userId", id)
					delete(state.users, id)
				}
			}
			stateMu.Unlock()
		}
	}()
}

func (db *InMemClient) CreateUser() string {
	newId := uuid.New().String()
	stateMu.Lock()
	state.users[newId] = user{
		tickets:     defaultTickets,
		lastUpdated: time.Now(),
	}
	stateMu.Unlock()
	return newId
}

func (db *InMemClient) GetAllByUserSplitByStatus(id string) (todo []models.Ticket, inProgress []models.Ticket, done []models.Ticket, err error) {
	stateMu.RLock()
	user, ok := state.users[id]
	stateMu.RUnlock()
	if !ok {
		err = ErrUserNotFound
		return
	}

	for _, t := range user.tickets {
		switch t.Status {
		case models.StatusTodo:
			todo = append(todo, t)
		case models.StatusInProgress:
			inProgress = append(inProgress, t)
		case models.StatusDone:
			done = append(done, t)
		}
	}

	return
}

func (db *InMemClient) GetAllByUser(id string) ([]models.Ticket, error) {
	stateMu.RLock()
	users, ok := state.users[id]
	stateMu.RUnlock()
	if !ok {
		return nil, ErrUserNotFound
	}

	return users.tickets, nil
}

func (db *InMemClient) AddToUser(id string, ticket models.Ticket) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	u, ok := state.users[id]
	if !ok {
		return ErrUserNotFound
	}

	u.tickets = append(u.tickets, ticket)
	u.lastUpdated = time.Now()
	state.users[id] = u

	return nil
}

func (db *InMemClient) DeleteUserByID(id string) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	delete(state.users, id)
	return nil
}

func (db *InMemClient) DeleteTodoByUserAndTodoId(userId, todoId string) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	found := -1

	u, ok := state.users[userId]
	if !ok {
		return ErrUserNotFound
	}

	for i, todo := range u.tickets {
		if todo.Id == todoId {
			found = i
			break
		}
	}

	if found < 0 {
		return fmt.Errorf("element not found")
	}

	u.tickets = slices.Delete(u.tickets, found, found+1)
	u.lastUpdated = time.Now()
	state.users[userId] = u

	return nil
}

func (db *InMemClient) UpdateUser(userId string, tickets []models.Ticket) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	u, ok := state.users[userId]
	if !ok {
		return ErrUserNotFound
	}

	u.tickets = tickets
	u.lastUpdated = time.Now()
	state.users[userId] = u

	return nil
}
