package logger

import (
	"encoding/json"
	"fmt"

	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/httputil"
	"github.com/MurmurationsNetwork/MurmurationsServices/pkg/logger"
)

type respond struct {
	Data geoInfo `json:"data,omitempty"`
}

type geoInfo struct {
	City    string  `json:"city,omitempty"`
	Country string  `json:"country,omitempty"`
	Lat     float64 `json:"lat,omitempty"`
	Lon     float64 `json:"lon,omitempty"`
}

func getGeoInfo(ip string) *geoInfo {
	bytes, err := httputil.GetByte(
		fmt.Sprintf("http://geoip-app:8080/city/%s", ip),
	)
	if err != nil {
		logger.Error(
			fmt.Sprintf(
				"Error when trying get http://geoip-app:8080/city/%s",
				ip,
			),
			err,
		)
		return &geoInfo{}
	}

	var respondData respond

	err = json.Unmarshal(bytes, &respondData)
	if err != nil {
		logger.Error("Error when trying to unmarshal GeoInfo respond data", err)
		return &geoInfo{}
	}

	return &respondData.Data
}
