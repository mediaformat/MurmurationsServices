package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MurmurationsNetwork/MurmurationsServices/services/dataproxy/internal/repository/db"
)

type ProfilesHandler interface {
	Get(c *gin.Context)
}

type profilesHandler struct {
	profileRepository db.ProfileRepository
}

func NewProfilesHandler(
	profileRepository db.ProfileRepository,
) ProfilesHandler {
	return &profilesHandler{
		profileRepository: profileRepository,
	}
}

func (handler *profilesHandler) Get(c *gin.Context) {
	profileID := c.Param("profileID")
	profile, err := handler.profileRepository.GetProfile(profileID)
	if err != nil {
		c.JSON(http.StatusNotFound, err)
		return
	}

	// remove id, __v, cuid, oid, node_id, is_posted
	delete(profile, "_id")
	delete(profile, "__v")
	delete(profile, "cuid")
	delete(profile, "oid")
	delete(profile, "node_id")
	delete(profile, "is_posted")
	delete(profile, "source_data_hash")
	// remove batch_id for batch import
	delete(profile, "batch_id")

	c.JSON(http.StatusOK, profile)
}
