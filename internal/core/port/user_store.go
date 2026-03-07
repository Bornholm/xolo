package port

import (
	"context"

	"github.com/bornholm/xolo/internal/core/model"
)

type UserStore interface {
	// FindOrCreateUser searches for a User in the store by its provider/subject unique tuple and returns
	// it if it exists, or create a new one otherwise
	FindOrCreateUser(ctx context.Context, provider, subject string) (model.User, error)

	// GetUserByID finds a user by its ID, or returns ErrNotFound if not found
	GetUserByID(ctx context.Context, userID model.UserID) (model.User, error)

	// QueryUsers returns a paginated list of users
	QueryUsers(ctx context.Context, opts QueryUsersOptions) ([]model.User, error)

	// SaveUser saves a user in the store
	SaveUser(ctx context.Context, user model.User) error

	// FindAuthToken searches for an AuthToken by its value, or returns ErrNotFound if not found
	FindAuthToken(ctx context.Context, token string) (model.AuthToken, error)

	// GetUserAuthTokens returns all the AuthToken associated to a User
	GetUserAuthTokens(ctx context.Context, userID model.UserID) ([]model.AuthToken, error)

	// CreateAuthToken creates a new AuthToken for a User
	CreateAuthToken(ctx context.Context, token model.AuthToken) error

	// DeleteAuthToken deletes an AuthToken by its ID
	DeleteAuthToken(ctx context.Context, tokenID model.AuthTokenID) error
}

type QueryUsersOptions struct {
	Page  *int
	Limit *int

	// Filters

	// Users with specific roles
	Roles []string

	// Active/inactive users
	Active *bool
}
