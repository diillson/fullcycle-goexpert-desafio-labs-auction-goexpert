package user_controller

import (
	"context"
	"fullcycle-auction_go/configuration/rest_err"
	"fullcycle-auction_go/internal/infra/api/web/validation"
	"fullcycle-auction_go/internal/usecase/user_usecase"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (u *UserController) CreateUser(c *gin.Context) {
	var userInputDTO user_usecase.UserCreateInputDTO

	if err := c.ShouldBindJSON(&userInputDTO); err != nil {
		restErr := validation.ValidateErr(err)
		c.JSON(restErr.Code, restErr)
		return
	}

	userOutput, err := u.userUseCase.CreateUser(context.Background(), userInputDTO)
	if err != nil {
		restErr := rest_err.ConvertError(err)
		c.JSON(restErr.Code, restErr)
		return
	}

	c.JSON(http.StatusCreated, userOutput)
}
