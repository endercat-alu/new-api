package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetRelayCaptureStatus(c *gin.Context) {
	common.ApiSuccess(c, service.GetRelayCaptureState())
}

func StartRelayCapture(c *gin.Context) {
	common.ApiSuccess(c, service.StartRelayCapture())
}

func StopRelayCapture(c *gin.Context) {
	common.ApiSuccess(c, service.StopRelayCapture())
}

func GetRelayCaptureRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	records := service.ListRelayCaptureRecords(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	pageInfo.SetTotal(records.Total)
	pageInfo.SetItems(records.Items)
	common.ApiSuccess(c, pageInfo)
}

func ClearRelayCaptureRecords(c *gin.Context) {
	common.ApiSuccess(c, service.ClearRelayCaptureRecords())
}
