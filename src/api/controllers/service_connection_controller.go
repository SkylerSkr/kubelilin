package controllers

import (
	"github.com/yoyofx/yoyogo/web/context"
	"github.com/yoyofx/yoyogo/web/mvc"
	"kubelilin/api/req"
	"kubelilin/domain/business/app"
)

type ServiceConnectionController struct {
	svc app.ServiceConnectionService
}

func (controller *ServiceConnectionController) CreateServiceConnection(ctx *context.HttpContext, request *req.ServiceConnectionReq) mvc.ApiResult {

	userInfo := req.GetUserInfo(ctx)
	request.TenantID = userInfo.TenantID
	res, err := controller.svc.CreateServiceConnection(request)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}

func (controller *ServiceConnectionController) UpdateServiceConnection(req *req.ServiceConnectionReq) mvc.ApiResult {
	res, err := controller.svc.UpdateServiceConnection(req)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}

func (controller *ServiceConnectionController) QueryServiceConnections(ctx *context.HttpContext) mvc.ApiResult {
	userInfo := req.GetUserInfo(ctx)
	res, err := controller.svc.QueryServiceConnectionInfo(userInfo.TenantID)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}
