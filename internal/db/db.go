package db

import (
	"fmt"
	"slices"
	"time"

	"github.com/JamesTiberiusKirk/lambdaban/internal/models"
	"github.com/google/uuid"
)

var (
	defaultTickets = []models.Ticket{
		{
			Id:            uuid.New().String(),
			Title:         "Test 1",
			Description:   "this is a test Description",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Status:        "todo",
		},
		{
			Id:            uuid.New().String(),
			Title:         "Test 2",
			Description:   "this is a test Description",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Status:        "todo",
		},
		{
			Id:            uuid.New().String(),
			Title:         "Test 3",
			Description:   "this is a test Description",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Status:        "todo",
		},
		{
			Id:            uuid.New().String(),
			Title:         "Test 4",
			Description:   "this is a test Description",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Status:        "todo",
		},
		{
			Id:            uuid.New().String(),
			Title:         "Test 5",
			Description:   "this is a test Description",
			CreatedAt:     time.Now(),
			LastUpdatedAt: time.Now(),
			Status:        "todo",
		},
	}
)

type globalState struct {
	tickets map[string][]models.Ticket
}

var state globalState

type InMemClient struct {
}

func NewInMemClient() *InMemClient {
	state = globalState{
		tickets: map[string][]models.Ticket{},
	}
	return &InMemClient{}
}

var (
	ErrUserNotFound = fmt.Errorf("user not found")
)

func (db *InMemClient) CreateUser() string {
	newId := uuid.New().String()

	state.tickets[newId] = defaultTickets

	return newId
}
func (db *InMemClient) GetAllByUserSplitByStatus(id string) (todo []models.Ticket, inProgress []models.Ticket, done []models.Ticket, err error) {
	ts, ok := state.tickets[id]
	if !ok {
		err = ErrUserNotFound
		return
	}

	for _, t := range ts {
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
	todos, ok := state.tickets[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return todos, nil
}

func (db *InMemClient) AddToUser(id string, todo models.Ticket) error {
	_, ok := state.tickets[id]
	if !ok {
		state.tickets[id] = []models.Ticket{}
	}

	state.tickets[id] = append(state.tickets[id], todo)
	return nil
}

func (db *InMemClient) DeleteUserByID(id string) error {
	delete(state.tickets, id)
	return nil
}

func (db *InMemClient) DeleteTodoByUserAndTodoId(userId, todoId string) error {
	found := -1

	todos, ok := state.tickets[userId]
	if !ok {
		return ErrUserNotFound
	}

	for i, todo := range todos {
		if todo.Id == todoId {
			found = i
		}
	}

	if found < 0 {
		return fmt.Errorf("element not found")
	}

	state.tickets[userId] = slices.Delete(todos, found, found+1)
	return nil
}

func (db *InMemClient) UpdateUser(userId string, tickets []models.Ticket) error {
	state.tickets[userId] = tickets
	return nil
}
