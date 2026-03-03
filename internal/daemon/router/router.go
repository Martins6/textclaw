package router

import (
	"database/sql"

	"github.com/Martins6/textclaw/internal/database"
)

type Router struct {
	db *database.DB
}

func New(db *database.DB) *Router {
	return &Router{db: db}
}

func (r *Router) Lookup(contactID string) (workspaceID string, err error) {
	contact, err := database.GetContact(r.db, contactID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrContactNotFound
		}
		return "", err
	}
	return contact.WorkspaceID, nil
}

var ErrContactNotFound = &ContactNotFoundError{}

type ContactNotFoundError struct{}

func (e *ContactNotFoundError) Error() string {
	return "contact not found"
}
