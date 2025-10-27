package controllers

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	helper "github.com/lintang-b-s/waze-traffic-scraper/pkg/http/router/routerhelper"
	"go.uber.org/zap"
)

type wazeAPI struct {
	trafficService TrafficService
	log            *zap.Logger
}

func New(trafficService TrafficService, log *zap.Logger) *wazeAPI {
	return &wazeAPI{
		trafficService: trafficService,
		log:            log,
	}
}

func (api *wazeAPI) Routes(group *helper.RouteGroup) {
	group.GET("/traffic", api.traffic)
}

func (api *wazeAPI) traffic(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var (
		err error
	)

	traffics, err := api.trafficService.GetRealtimeTraffic()
	if err != nil {
		api.getStatusCode(w, r, err)
		return
	}

	headers := make(http.Header)

	if err := api.writeJSON(w, http.StatusOK, envelope{"data": NewTrafficResponse(traffics)}, headers); err != nil {
		api.ServerErrorResponse(w, r, err)
		return
	}
}
