package db

import (
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
