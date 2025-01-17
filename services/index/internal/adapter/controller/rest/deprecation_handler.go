package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/jsonapi"
)

type DeprecationHandler interface {
	DeprecationV1(c *gin.Context)
}

type deprecationHandler struct {
}

func NewDeprecationHandler() DeprecationHandler {
	return &deprecationHandler{}
}

func (handler *deprecationHandler) DeprecationV1(c *gin.Context) {
	errors := jsonapi.NewError(
		[]string{"Gone"},
		[]string{
			"The v1 API has been deprecated. Please use the v2 API instead: https://app.swaggerhub.com/apis-docs/MurmurationsNetwork/IndexAPI/2.0.0",
		},
		nil,
		[]int{http.StatusGone},
	)
	res := jsonapi.Response(nil, errors, nil, nil)
	c.JSON(errors[0].Status, res)
}
