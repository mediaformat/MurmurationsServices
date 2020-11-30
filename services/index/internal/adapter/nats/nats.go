package nats

import (
	"os"

	"github.com/MurmurationsNetwork/MurmurationsServices/common/logger"
	"github.com/MurmurationsNetwork/MurmurationsServices/common/nats"
)

func Init() {
	err := nats.NewClient(os.Getenv("NATS_CLUSTER_ID"), os.Getenv("NATS_CLIENT_ID"), os.Getenv("NATS_URL"))
	if err != nil {
		logger.Panic("error when trying to connect nats", err)
	}
}