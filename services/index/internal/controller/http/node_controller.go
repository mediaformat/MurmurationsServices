package http

import (
	"net/http"

	"github.com/MurmurationsNetwork/MurmurationsServices/common/resterr"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/index/internal/domain/node"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/index/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	NodeController nodeControllerInterface = &nodeController{}
)

type nodeControllerInterface interface {
	Add(c *gin.Context)
	Get(c *gin.Context)
	Search(c *gin.Context)
	Delete(c *gin.Context)
}

type nodeController struct{}

func (cont *nodeController) getNodeId(params gin.Params) (string, resterr.RestErr) {
	nodeId, found := params.Get("nodeId")
	if !found {
		return "", resterr.NewBadRequestError("invalid node id")
	}
	return nodeId, nil
}

func (cont *nodeController) Add(c *gin.Context) {
	var node node.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		restErr := resterr.NewBadRequestError("invalid json body")
		c.JSON(restErr.Status(), restErr)
		return
	}

	result, err := service.NodeService.AddNode(node)
	if err != nil {
		c.JSON(err.Status(), err)
		return
	}

	c.JSON(http.StatusOK, result.Marshall())
}

func (cont *nodeController) Get(c *gin.Context) {
}

func (cont *nodeController) Search(c *gin.Context) {
	var query node.NodeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		restErr := resterr.NewBadRequestError(err.Error())
		c.JSON(restErr.Status(), restErr)
		return
	}

	searchRes, err := service.NodeService.SearchNode(&query)
	if err != nil {
		c.JSON(err.Status(), err)
		return
	}

	c.JSON(http.StatusOK, searchRes.Marshall())
}

func (cont *nodeController) Delete(c *gin.Context) {
	nodeId, err := cont.getNodeId(c.Params)
	if err != nil {
		c.JSON(err.Status(), err)
		return
	}

	err = service.NodeService.DeleteNode(nodeId)
	if err != nil {
		c.JSON(err.Status(), err)
		return
	}

	c.JSON(http.StatusOK, nil)
}