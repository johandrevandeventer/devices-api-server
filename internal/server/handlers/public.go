package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/johandrevandeventer/devices-api-server/internal/config"
	serverutils "github.com/johandrevandeventer/devices-api-server/internal/server/utils"
)

func HealthHandler(c *gin.Context) {
	cfg := config.GetConfig()
	data := fmt.Sprintf("Service is running: %s", cfg.System.AppName)
	serverutils.WriteJSON(c, http.StatusOK, "OK", data)
}
