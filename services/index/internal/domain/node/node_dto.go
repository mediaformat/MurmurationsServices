package node

import (
	"github.com/MurmurationsNetwork/MurmurationsServices/common/constant"
	"github.com/MurmurationsNetwork/MurmurationsServices/common/resterr"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Node struct {
	ID            primitive.ObjectID  `json:"_id" bson:"_id,omitempty"`
	NodeID        string              `json:"nodeId" bson:"nodeId,omitempty"`
	ProfileUrl    string              `json:"profileUrl" bson:"profileUrl,omitempty"`
	ProfileHash   string              `json:"profileHash" bson:"profileHash,omitempty"`
	LinkedSchemas []string            `json:"linkedSchemas" bson:"linkedSchemas,omitempty"`
	Status        constant.NodeStatus `json:"status" bson:"status,omitempty"`
	LastValidated int64               `json:"lastValidated" bson:"lastValidated,omitempty"`
	FailedReasons *[]string           `json:"failedReasons" bson:"failedReasons,omitempty"`
	Version       *int32              `json:"version" bson:"version,omitempty"`
}

func (node *Node) Validate() resterr.RestErr {
	if node.ProfileUrl == "" {
		return resterr.NewBadRequestError("profileUrl parameter is missing.")
	}

	if len(node.LinkedSchemas) == 0 {
		return resterr.NewBadRequestError("linkedSchemas parameter is missing.")
	}

	return nil
}

type Nodes []Node

type NodeQuery struct {
	Schema        string `form:"schema"`
	LastValidated int64  `form:"lastValidated"`
}
