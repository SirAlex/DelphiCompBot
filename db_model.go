package main

import (
	"github.com/jinzhu/gorm"
	"time"
)

type User struct {
	ID          int `gorm:"primary_key;AUTO_INCREMENT:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `sql:"index"`
	RBInfo      RuBoardInfo
	RBInfoID    uint
	UserID      int    // Telegram user ID
	UserName    string // Telegram user name
	ChatID      int64  // Chat ID with user
	Registered  bool   // User pass registration procedure
	LastCommand string // Last command to track multistep commands
}

// RU-Board user information
type RuBoardInfo struct {
	gorm.Model
	Login            string     `gorm:"size:50"` // Forum login
	RegisteredAt     *time.Time // Registration date
	ConfirmationCode string     // Generated registration confirmation code
	ConfirmTryCount  int        // Confirmation try count
	RegPoints        int
	WarezPoints      int
	BonusPoints      int
	TotalPoints      int
	PointsCheckedAt  *time.Time // When last message count check was performed
}

func (i *RuBoardInfo) RecalcPoints() {
	i.TotalPoints = i.RegPoints + i.WarezPoints + i.BonusPoints
}
