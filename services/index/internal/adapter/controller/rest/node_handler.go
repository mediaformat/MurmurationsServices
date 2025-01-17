package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/constant"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/dateutil"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/jsonapi"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/logger"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/validatenode"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/index/config"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/index/internal/entity/query"
	"github.com/MurmurationsNetwork/MurmurationsServices/services/index/internal/usecase"
)

type NodeHandler interface {
	Add(c *gin.Context)
	Get(c *gin.Context)
	Search(c *gin.Context)
	Delete(c *gin.Context)
	AddSync(c *gin.Context)
	Validate(c *gin.Context)
	Export(c *gin.Context)
	GetNodes(c *gin.Context)
}

type nodeHandler struct {
	nodeUsecase usecase.NodeUsecase
}

func NewNodeHandler(nodeService usecase.NodeUsecase) NodeHandler {
	return &nodeHandler{
		nodeUsecase: nodeService,
	}
}

func (handler *nodeHandler) getNodeID(
	params gin.Params,
) (string, []jsonapi.Error) {
	nodeID, found := params.Get("nodeID")
	if !found {
		return "", jsonapi.NewError(
			[]string{"Invalid Node Id"},
			[]string{"The `node_id` is invalid."},
			nil,
			[]int{http.StatusBadRequest},
		)
	}
	return nodeID, nil
}

func (handler *nodeHandler) Add(c *gin.Context) {
	var node nodeDTO
	if err := c.ShouldBindJSON(&node); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	if err := node.Validate(); err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	result, err := handler.nodeUsecase.AddNode(node.toEntity())
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	res := jsonapi.Response(handler.ToAddNodeVO(result), nil, nil, nil)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) Get(c *gin.Context) {
	nodeID, err := handler.getNodeID(c.Params)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	node, err := handler.nodeUsecase.GetNode(nodeID)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	if node.Status == constant.NodeStatus.PostFailed {
		meta := jsonapi.NewMeta(
			"The system will automatically re-post the node, please check back in a minute.",
			"",
			"",
		)
		res := jsonapi.Response(handler.ToGetNodeVO(node), nil, nil, meta)
		c.JSON(http.StatusOK, res)
		return
	}

	if node.Status == constant.NodeStatus.ValidationFailed {
		meta := jsonapi.NewMeta("", node.ID, node.ProfileURL)
		errors := *node.FailureReasons
		res := jsonapi.Response(nil, errors, nil, meta)
		c.JSON(errors[0].Status, res)
		return
	}

	res := jsonapi.Response(handler.ToGetNodeVO(node), nil, nil, nil)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) Search(c *gin.Context) {
	// return error if there is an invalid query
	// get the fields from query.EsQuery
	fields := [...]string{
		"schema",
		"last_updated",
		"lat",
		"lon",
		"range",
		"locality",
		"region",
		"country",
		"status",
		"tags",
		"tags_filter",
		"tags_exact",
		"primary_url",
		"page",
		"page_size",
	}
	queryFields := c.Request.URL.Query()
	var (
		invalidQueryTitles, invalidQueryDetails []string
		invalidQuerySources                     [][]string
		invalidQueryStatus                      []int
	)
	for fieldName := range queryFields {
		found := false
		for _, validFieldName := range fields {
			if fieldName == validFieldName {
				found = true
				break
			}
		}
		if !found {
			invalidQueryTitles = append(
				invalidQueryTitles,
				"Invalid Query Parameter",
			)
			invalidQueryDetails = append(
				invalidQueryDetails,
				fmt.Sprintf(
					"The following query parameter is not valid: %v",
					fieldName,
				),
			)
			invalidQuerySources = append(
				invalidQuerySources,
				[]string{"parameter", fieldName},
			)
			invalidQueryStatus = append(
				invalidQueryStatus,
				http.StatusBadRequest,
			)
		}
	}

	if len(invalidQueryTitles) != 0 {
		errors := jsonapi.NewError(
			invalidQueryTitles,
			invalidQueryDetails,
			invalidQuerySources,
			invalidQueryStatus,
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	var esQuery query.EsQuery
	if err := c.ShouldBindQuery(&esQuery); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	if esQuery.Page*esQuery.PageSize > 10000 {
		errors := jsonapi.NewError(
			[]string{"Max Results Exceeded"},
			[]string{
				"No more than 10,000 results can be returned. Refine your query so it will return less but more relevant results.",
			},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	searchResult, err := handler.nodeUsecase.Search(&esQuery)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	// restrict the last page to the page of 10,000 results (ES limitation)
	totalPage := 10000 / esQuery.PageSize
	message := "No more than 10,000 results can be returned. Refine your query so it will return less but more relevant results."
	if totalPage >= searchResult.TotalPages {
		totalPage = searchResult.TotalPages
		message = ""
	}
	// edge case: page = 0 or larger than total page - response no data
	if searchResult.TotalPages == 0 || esQuery.Page > searchResult.TotalPages {
		res := jsonapi.Response(searchResult.Result, nil, nil, nil)
		c.JSON(http.StatusOK, res)
		return
	}
	meta := jsonapi.NewSearchMeta(
		message,
		searchResult.NumberOfResults,
		searchResult.TotalPages,
	)
	links := jsonapi.NewLinks(c, esQuery.Page, totalPage)
	res := jsonapi.Response(searchResult.Result, nil, links, meta)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) Delete(c *gin.Context) {
	if c.Params.ByName("nodeID") == "" {
		errors := jsonapi.NewError(
			[]string{"Missing Path Parameter"},
			[]string{"The `node_id` path parameter is missing."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	nodeID, err := handler.getNodeID(c.Params)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	profileURL, err := handler.nodeUsecase.Delete(nodeID)
	if err != nil {
		meta := jsonapi.NewMeta("", nodeID, profileURL)
		res := jsonapi.Response(nil, err, nil, meta)
		c.JSON(err[0].Status, res)
		return
	}

	deleteTTL := dateutil.FormatSeconds(config.Conf.TTL.DeletedTTL)

	meta := jsonapi.NewMeta(
		fmt.Sprintf(
			"The Index has recorded as deleted the profile that was previously "+
				"posted at: %s -- It will be completely removed from the index in %s.",
			profileURL,
			deleteTTL,
		),
		"",
		"",
	)
	res := jsonapi.Response(nil, nil, nil, meta)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) AddSync(c *gin.Context) {
	var node nodeDTO
	if err := c.ShouldBindJSON(&node); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	if err := node.Validate(); err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	result, err := handler.nodeUsecase.AddNode(node.toEntity())
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	// try the 1st time in 1 second, 2nd time in 2 seconds, 3rd in 4, 4th in 8, 5th in 16 seconds.
	waitInterval := 1 * time.Second
	retries := 5

	for retries != 0 {
		nodeInfo, err := handler.nodeUsecase.GetNode(result.ID)
		if err != nil {
			res := jsonapi.Response(nil, err, nil, nil)
			c.JSON(err[0].Status, res)
			return
		}

		if nodeInfo.Status == constant.NodeStatus.PostFailed {
			meta := jsonapi.NewMeta(
				"The system will automatically re-post the node, please check back in a minute.",
				"",
				"",
			)
			res := jsonapi.Response(handler.ToGetNodeVO(result), nil, nil, meta)
			c.JSON(http.StatusOK, res)
			return
		}

		if nodeInfo.Status == constant.NodeStatus.ValidationFailed {
			meta := jsonapi.NewMeta("", nodeInfo.ID, nodeInfo.ProfileURL)
			errors := *nodeInfo.FailureReasons
			res := jsonapi.Response(nil, errors, nil, meta)
			c.JSON(errors[0].Status, res)
			return
		}

		if nodeInfo.Status == constant.NodeStatus.Posted ||
			nodeInfo.Status == constant.NodeStatus.Deleted {
			res := jsonapi.Response(
				handler.ToGetNodeVO(nodeInfo),
				nil,
				nil,
				nil,
			)
			c.JSON(http.StatusOK, res)
			return
		}

		time.Sleep(waitInterval)
		waitInterval *= 2
		retries--
	}

	// if server can't get the node with posted or failed information, return the node id for user to get the node in the future.
	res := jsonapi.Response(handler.ToAddNodeVO(result), nil, nil, nil)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) Validate(c *gin.Context) {
	var node interface{}

	if err := c.ShouldBindJSON(&node); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	jsonString, err := json.Marshal(node)
	if err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The body of the JSON document submitted is malformed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	linkedSchemas, ok := getLinkedSchemas(node)
	if !ok {
		errors := jsonapi.NewError(
			[]string{"Missing Required Property"},
			[]string{"The `linked_schemas` property is required."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	// Validate against the default schema.
	linkedSchemas = append(linkedSchemas, "default-v2.0.0")

	// Validate against schemes specify inside the profile data.
	result := validatenode.ValidateAgainstSchemas(
		config.Conf.Library.InternalURL,
		linkedSchemas,
		string(jsonString),
		"string",
	)
	if !result.Valid {
		message := "Failed to validate against schemas: " + strings.Join(
			result.ErrorMessages,
			" ",
		)
		logger.Info(message)
		errors := jsonapi.NewError(
			result.ErrorMessages,
			result.Details,
			result.Sources,
			result.ErrorStatus,
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	meta := jsonapi.NewMeta(
		"The submitted profile was validated successfully to its linked schemas.",
		"",
		"",
	)
	res := jsonapi.Response(nil, nil, nil, meta)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) Export(c *gin.Context) {
	// return error if there is an invalid query
	// get the fields from query.EsQuery
	fields := [...]string{"schema", "page_size", "search_after"}
	queryFields := c.Request.URL.Query()
	var (
		invalidQueryTitles, invalidQueryDetails []string
		invalidQuerySources                     [][]string
		invalidQueryStatus                      []int
	)
	for fieldName := range queryFields {
		found := false
		for _, validFieldName := range fields {
			if fieldName == validFieldName {
				found = true
				break
			}
		}
		if !found {
			invalidQueryTitles = append(
				invalidQueryTitles,
				"Invalid Query Parameter",
			)
			invalidQueryDetails = append(
				invalidQueryDetails,
				fmt.Sprintf(
					"The following query parameter is not valid: %v",
					fieldName,
				),
			)
			invalidQuerySources = append(
				invalidQuerySources,
				[]string{"parameter", fieldName},
			)
			invalidQueryStatus = append(
				invalidQueryStatus,
				http.StatusBadRequest,
			)
		}
	}

	if len(invalidQueryTitles) != 0 {
		errors := jsonapi.NewError(
			invalidQueryTitles,
			invalidQueryDetails,
			invalidQuerySources,
			invalidQueryStatus,
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	var esQuery query.EsBlockQuery
	if err := c.ShouldBindJSON(&esQuery); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	// set default page_size for esQuery
	if esQuery.PageSize == 0 {
		esQuery.PageSize = 100
	}

	searchResult, err := handler.nodeUsecase.Export(&esQuery)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	meta := jsonapi.NewBlockSearchMeta(searchResult.Sort)
	res := jsonapi.Response(searchResult.Result, nil, nil, meta)
	c.JSON(http.StatusOK, res)
}

func (handler *nodeHandler) GetNodes(c *gin.Context) {
	// return error if there is an invalid query
	// get the fields from query.EsQuery
	fields := [...]string{
		"schema",
		"last_updated",
		"lat",
		"lon",
		"range",
		"locality",
		"region",
		"country",
		"status",
		"tags",
		"tags_filter",
		"tags_exact",
		"primary_url",
		"page",
		"page_size",
	}
	queryFields := c.Request.URL.Query()
	var (
		invalidQueryTitles, invalidQueryDetails []string
		invalidQuerySources                     [][]string
		invalidQueryStatus                      []int
	)
	for fieldName := range queryFields {
		found := false
		for _, validFieldName := range fields {
			if fieldName == validFieldName {
				found = true
				break
			}
		}
		if !found {
			invalidQueryTitles = append(
				invalidQueryTitles,
				"Invalid Query Parameter",
			)
			invalidQueryDetails = append(
				invalidQueryDetails,
				fmt.Sprintf(
					"The following query parameter is not valid: %v",
					fieldName,
				),
			)
			invalidQuerySources = append(
				invalidQuerySources,
				[]string{"parameter", fieldName},
			)
			invalidQueryStatus = append(
				invalidQueryStatus,
				http.StatusBadRequest,
			)
		}
	}

	if len(invalidQueryTitles) != 0 {
		errors := jsonapi.NewError(
			invalidQueryTitles,
			invalidQueryDetails,
			invalidQuerySources,
			invalidQueryStatus,
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	var esQuery query.EsQuery
	if err := c.ShouldBindQuery(&esQuery); err != nil {
		errors := jsonapi.NewError(
			[]string{"JSON Error"},
			[]string{"The JSON document submitted could not be parsed."},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	if esQuery.Page*esQuery.PageSize > 10000 {
		errors := jsonapi.NewError(
			[]string{"Max Results Exceeded"},
			[]string{
				"No more than 10,000 results can be returned. Refine your query so it will return less but more relevant results.",
			},
			nil,
			[]int{http.StatusBadRequest},
		)
		res := jsonapi.Response(nil, errors, nil, nil)
		c.JSON(errors[0].Status, res)
		return
	}

	searchResult, err := handler.nodeUsecase.GetNodes(&esQuery)
	if err != nil {
		res := jsonapi.Response(nil, err, nil, nil)
		c.JSON(err[0].Status, res)
		return
	}

	// restrict the last page to the page of 10,000 results (ES limitation)
	totalPage := 10000 / esQuery.PageSize
	message := "No more than 10,000 results can be returned. Refine your query so it will return less but more relevant results."
	if totalPage >= searchResult.TotalPages {
		totalPage = searchResult.TotalPages
		message = ""
	}
	// edge case: page = 0 or larger than total page - response no data
	if searchResult.TotalPages == 0 || esQuery.Page > searchResult.TotalPages {
		res := jsonapi.Response(searchResult.Result, nil, nil, nil)
		c.JSON(http.StatusOK, res)
		return
	}

	meta := jsonapi.NewSearchMeta(
		message,
		searchResult.NumberOfResults,
		searchResult.TotalPages,
	)
	links := jsonapi.NewLinks(c, esQuery.Page, totalPage)
	res := jsonapi.Response(searchResult.Result, nil, links, meta)
	c.JSON(http.StatusOK, res)
}

func getLinkedSchemas(data interface{}) ([]string, bool) {
	json, ok := data.(map[string]interface{})
	if !ok {
		return nil, false
	}
	_, ok = json["linked_schemas"]
	if !ok {
		return nil, false
	}
	arrInterface, ok := json["linked_schemas"].([]interface{})
	if !ok {
		return nil, false
	}

	var linkedSchemas = make([]string, 0)

	for _, data := range arrInterface {
		linkedSchema, ok := data.(string)
		if !ok {
			return nil, false
		}
		linkedSchemas = append(linkedSchemas, linkedSchema)
	}

	return linkedSchemas, true
}
