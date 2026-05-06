package models

import "github.com/google/uuid"

type ProfileServiceRole struct {
	ServiceID   uuid.UUID
	ServiceName string
	Role        string
}
