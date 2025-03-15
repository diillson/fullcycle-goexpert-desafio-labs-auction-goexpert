package user_usecase

import (
	"context"
	"fullcycle-auction_go/internal/entity/user_entity"
	"fullcycle-auction_go/internal/internal_error"
	"github.com/google/uuid"
)

type UserCreateInputDTO struct {
	Name string `json:"name" binding:"required,min=3"`
}

type UserCreateOutputDTO struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Adicionar ao UserUseCaseInterface
// Interface já existente precisa ser atualizada para incluir o novo método
// Adicione esta linha na definição da interface em find_user_usecase.go:
//
// CreateUser(ctx context.Context, input UserCreateInputDTO) (*UserCreateOutputDTO, *internal_error.InternalError)

func (u *UserUseCase) CreateUser(
	ctx context.Context, input UserCreateInputDTO) (*UserCreateOutputDTO, *internal_error.InternalError) {

	// Criar uma nova entidade de usuário
	user := &user_entity.User{
		Id:   uuid.New().String(),
		Name: input.Name,
	}

	// Inserir no banco de dados
	if err := u.UserRepository.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Retornar DTO de saída
	return &UserCreateOutputDTO{
		Id:   user.Id,
		Name: user.Name,
	}, nil
}
