package app

import (
	"github.com/MurmurationsNetwork/MurmurationsServices/indexer/internal/controllers/nodes"
	"github.com/MurmurationsNetwork/MurmurationsServices/indexer/internal/controllers/ping"
)

func mapUrls() {
	router.GET("/ping", ping.Ping)

	router.POST("/nodes", nodes.Add)
	router.GET("/nodes", nodes.Search)
	router.DELETE("/nodes/:node_id", nodes.Delete)
}
