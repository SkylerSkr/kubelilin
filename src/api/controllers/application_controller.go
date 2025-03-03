package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/yoyofx/yoyogo/web/binding"
	"github.com/yoyofx/yoyogo/web/context"
	"github.com/yoyofx/yoyogo/web/mvc"
	requests2 "kubelilin/api/dto/requests"
	"kubelilin/domain/business/app"
	"kubelilin/domain/database/models"
	"kubelilin/domain/dto"
	"kubelilin/utils"
)

type ApplicationController struct {
	mvc.ApiController
	service         *app.ApplicationService
	pipelineService *app.PipelineService
}

func NewApplicationController(service *app.ApplicationService, pipelineService *app.PipelineService) *ApplicationController {
	return &ApplicationController{service: service, pipelineService: pipelineService}
}

// PostCreateApp new application.
func (c *ApplicationController) PostCreateApp(ctx *context.HttpContext, request *requests2.AppReq) mvc.ApiResult {
	userInfo := requests2.GetUserInfo(ctx)
	request.TenantID = userInfo.TenantID
	err, res := c.service.CreateApp(request)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}

// PutEditApp edit application information.
func (c *ApplicationController) PutEditApp(ctx *context.HttpContext, request *requests2.AppReq) mvc.ApiResult {
	userInfo := requests2.GetUserInfo(ctx)
	request.TenantID = userInfo.TenantID
	err, res := c.service.UpdateApp(request)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}

// GetAppList get application list by tenant id.
func (c *ApplicationController) GetAppList(ctx *context.HttpContext) mvc.ApiResult {
	request := requests2.AppReq{}
	_ = ctx.BindWithUri(&request)

	userInfo := requests2.GetUserInfo(ctx)
	request.TenantID = userInfo.TenantID
	err, res := c.service.QueryAppList(&request)
	fmt.Println(res.Data)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(res)
}

// GetAppLanguage get language for application
func (c *ApplicationController) GetAppLanguage() mvc.ApiResult {
	res := c.service.QueryAppCodeLanguage()
	return mvc.Success(res)
}

// GetAppLevel get level for application
func (c *ApplicationController) GetAppLevel() mvc.ApiResult {
	res := c.service.QueryAppLevel()
	return mvc.Success(res)
}

func (c *ApplicationController) GetDeployLevel() mvc.ApiResult {
	res := c.service.QueryDeployLevel()
	return mvc.Success(res)
}

func (c *ApplicationController) GetDeployLevelCounts(ctx *context.HttpContext) mvc.ApiResult {
	appId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("appid", "0"))
	res, _ := c.service.GetAppCountByDeployLevel(appId)
	return mvc.Success(res)
}

func (c *ApplicationController) GetProjectDeployLevelCounts(ctx *context.HttpContext) mvc.ApiResult {
	projectId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("projectId", "0"))
	res, _ := c.service.GetProjectCountByDeployLevel(projectId)
	return mvc.Success(res)
}

// GetGitRepo get git address for application
func (c *ApplicationController) GetGitRepo(ctx *context.HttpContext) mvc.ApiResult {
	userInfo := requests2.GetUserInfo(ctx)
	appName := ctx.Input.Query("appName")
	cvsRes, err := c.service.InitGitRepository(userInfo.TenantID, appName)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(cvsRes)
}

// GetInfo get application information
func (c *ApplicationController) GetInfo(ctx *context.HttpContext) mvc.ApiResult {
	appId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("appid", "0"))
	info, err := c.service.GetAppInfo(appId)
	if err != nil {
		return mvc.FailWithMsg(nil, err.Error())
	}
	return mvc.Success(info)
}

// GetGitBranches get git addresses & branches for pipeline
func (c *ApplicationController) GetGitBranches(ctx *context.HttpContext) mvc.ApiResult {
	appId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("appid", "0"))
	appInfo, _ := c.service.GetAppInfo(appId)
	token := ""
	if appInfo.SCID > 0 {
		scInfo, _ := c.service.GetServiceConnectionById(appInfo.SCID)
		var detail dto.ServiceConnectionDetails
		_ = json.Unmarshal([]byte(scInfo.Detail), &detail)
		token = detail.Token
	}

	if appInfo.Git != "" {
		names, _ := app.GetGitBranches(appInfo.Git, appInfo.SourceType, token)
		return mvc.Success(context.H{
			"git":      appInfo.Git,
			"branches": names,
		})
	}
	// appInfo.Git
	return mvc.Fail("no data")
}

func (c *ApplicationController) GetBuildImageByLanguageId(ctx *context.HttpContext) mvc.ApiResult {
	languageId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("languageId", "0"))
	list, err := c.pipelineService.GetBuildImageByLanguageId(languageId)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(list)
}

func (c *ApplicationController) GetBuildImageByLanguages(ctx *context.HttpContext) mvc.ApiResult {
	languageId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("languageId", "0"))
	aliasName := ctx.Input.QueryDefault("aliasName", "")
	list, err := c.pipelineService.GetBuildImageBy(aliasName, languageId)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(list)
}

func (c *ApplicationController) PostBuildImage(ctx *context.HttpContext) mvc.ApiResult {
	var request models.ApplicationLanguageCompile
	_ = ctx.BindWith(&request, binding.JSON)
	request.Status = 1
	err := c.pipelineService.AddOrEditBuildImage(request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(true)
}

func (c *ApplicationController) DeleteBuildImage(ctx *context.HttpContext) mvc.ApiResult {
	id, _ := utils.StringToUInt64(ctx.Input.QueryDefault("id", "0"))
	err := c.pipelineService.DeleteBuildImage(id)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(true)
}

// PostNewPipeline new pipeline only by name & id
func (c *ApplicationController) PostNewPipeline(req *requests2.AppNewPipelineReq) mvc.ApiResult {
	err, pipeline := c.pipelineService.NewPipeline(req)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(pipeline.ID)
}

// GetPipelines get pipeline list by application id.
func (c *ApplicationController) GetPipelines(ctx *context.HttpContext) mvc.ApiResult {
	appId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("appid", "0"))
	if appId == 0 {
		return mvc.Fail("没有找到应用")
	}
	pipelines, err := c.pipelineService.GetAppPipelines(appId)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(pipelines)
}

// PostEditPipeline Save pipeline information and DSL.
func (c *ApplicationController) PostEditPipeline(request *requests2.EditPipelineReq) mvc.ApiResult {
	err := c.pipelineService.UpdatePipeline(request)
	if err == nil {
		err = c.pipelineService.UpdateDSL(request)
	}
	if err != nil {
		return mvc.FailWithMsg(false, err.Error())
	}

	return mvc.Success(true)
}

// GetPipeline get pipeline frontend json by id.
func (c *ApplicationController) GetPipeline(ctx *context.HttpContext) mvc.ApiResult {
	pipelineId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("id", "0"))
	pipeline, err := c.pipelineService.GetPipelineById(pipelineId)
	if err != nil {
		return mvc.Fail("not found pipeline!")
	}
	return mvc.Success(pipeline)
}

func (c *ApplicationController) PostAbortPipeline(request *requests2.AbortPipelineReq) mvc.ApiResult {
	err := c.pipelineService.AbortPipeline(request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(true)
}

func (c *ApplicationController) PostRunPipeline(request *requests2.RunPipelineReq) mvc.ApiResult {
	taskId, err := c.pipelineService.RunPipeline(request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(taskId)
}

func (c *ApplicationController) PostPipelineStatus(request *requests2.PipelineStatusReq) mvc.ApiResult {
	err := c.pipelineService.UpdatePipelineStatus(request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(true)
}

func (c *ApplicationController) DeletePipeline(ctx *context.HttpContext) mvc.ApiResult {
	pipelineId, _ := utils.StringToUInt64(ctx.Input.QueryDefault("id", "0"))
	err := c.pipelineService.DeletePipeline(pipelineId)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(true)
}

func (c *ApplicationController) GetPipelineDetails(httpContext *context.HttpContext) mvc.ApiResult {
	var request requests2.PipelineDetailsReq
	_ = httpContext.BindWithUri(&request)
	job, err := c.pipelineService.GetDetails(&request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(job)
}

func (c *ApplicationController) GetPipelineLogs(httpContext *context.HttpContext) mvc.ApiResult {
	var request requests2.PipelineDetailsReq
	_ = httpContext.BindWithUri(&request)
	logs, err := c.pipelineService.GetLogs(&request)
	if err != nil {
		return mvc.Fail(err.Error())
	}
	return mvc.Success(logs)
}
