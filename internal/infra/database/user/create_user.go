package user

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/user_entity"
	"fullcycle-auction_go/internal/internal_error"
)

func (ur *UserRepository) CreateUser(
	ctx context.Context, user *user_entity.User) *internal_error.InternalError {

	userEntityMongo := &UserEntityMongo{
		Id:   user.Id,
		Name: user.Name,
	}

	_, err := ur.Collection.InsertOne(ctx, userEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert user", err)
		return internal_error.NewInternalServerError("Error trying to insert user")
	}

	return nil
}
