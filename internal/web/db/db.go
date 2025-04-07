package db

import (
	"fmt"
	"slices"

	"github.com/JamesTiberiusKirk/todoist/internal/models"
)

type globalState struct {
	todos map[string][]models.Todos
}

var state globalState

type InMemClient struct {
}

func NewInMemClient() *InMemClient {
	state = globalState{
		todos: map[string][]models.Todos{},
	}
	return &InMemClient{}
}

var (
	ErrUserNotFound = fmt.Errorf("user not found")
)

func (db *InMemClient) GetAllByUser(id string) ([]models.Todos, error) {
	todos, ok := state.todos[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	return todos, nil
}

func (db *InMemClient) AddToUser(id string, todo models.Todos) error {
	_, ok := state.todos[id]
	if !ok {
		state.todos[id] = []models.Todos{}
	}

	state.todos[id] = append(state.todos[id], todo)
	return nil
}

func (db *InMemClient) DeleteUserByID(id string) error {
	delete(state.todos, id)
	return nil
}

func (db *InMemClient) DeleteTodoByUserAndTodoId(userId, todoId string) error {
	found := -1

	todos, ok := state.todos[userId]
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

	state.todos[userId] = slices.Delete(todos, found, found+1)

	fmt.Println("deleting ", userId, found)

	return nil
}
