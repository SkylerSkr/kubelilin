package app

import (
	"errors"
	"gorm.io/gorm"
	requests2 "kubelilin/api/dto/requests"
	"kubelilin/domain/database/models"
	"kubelilin/domain/dto"
	"kubelilin/pkg/page"
	"kubelilin/utils"
	"strings"
)

type DevopsService struct {
	db *gorm.DB
}

func NewDevopsService(db *gorm.DB) *DevopsService {
	return &DevopsService{db: db}
}

func (service *DevopsService) CreateProject(requestProject *requests2.NewProject) error {
	var exitCount int64
	service.db.Model(&models.DevopsProjects{}).Where("tenant_id=? and name=?", requestProject.TenantID, requestProject.Name).Count(&exitCount)
	if exitCount > 0 {
		return errors.New("already have the same name project")
	}

	devProject := models.DevopsProjects{
		Name:         requestProject.Name,
		TenantID:     requestProject.TenantID,
		CreationTime: utils.TimeNowPtr(),
	}
	dpcRes := service.db.Model(&models.DevopsProjects{}).Create(&devProject)
	if dpcRes.Error != nil {
		return dpcRes.Error
	}
	for _, appId := range requestProject.AppIdList {
		devProjectApps := models.DevopsProjectsApps{
			ProjectID:     devProject.ID,
			ApplicationID: appId,
		}
		service.db.Model(&models.DevopsProjectsApps{}).Create(&devProjectApps)
	}

	return nil
}

func (service *DevopsService) EditProject(updateRequest *requests2.NewProject) error {
	devProject := models.DevopsProjects{
		ID:           updateRequest.ProjectId,
		Name:         updateRequest.Name,
		TenantID:     updateRequest.TenantID,
		CreationTime: utils.TimeNowPtr(),
	}
	err := service.db.Transaction(func(tx *gorm.DB) error {
		dpcRes := tx.Model(&models.DevopsProjects{}).Where("id=?", updateRequest.ProjectId).Updates(devProject)
		if dpcRes.Error != nil {
			return dpcRes.Error
		}
		dpcRes = tx.Model(&models.DevopsProjectsApps{}).Delete(&models.DevopsProjectsApps{}, "project_id=?", updateRequest.ProjectId)
		if dpcRes.Error == nil {
			for _, appId := range updateRequest.AppIdList {
				devProjectApps := models.DevopsProjectsApps{
					ProjectID:     devProject.ID,
					ApplicationID: appId,
				}
				tx.Model(&models.DevopsProjectsApps{}).Create(&devProjectApps)
			}
		}
		return nil
	})
	return err
}

func (service *DevopsService) DeleteProject(projectId uint64) error {
	//dpcRes := service.db.Model(&models.DevopsProjects{}).Where("id=?", projectId).Updates(devProject)
	return service.db.Exec("UPDATE devops_projects SET soft_del = 1 WHERE id=?", projectId).Error
}

func (service *DevopsService) GetProjectList(request *requests2.DevopsProjectReq) (error, *page.Page) {
	sql := `SELECT p.id,p.name 'project_name', p.creation_time, 
(SELECT count(1) FROM  devops_projects_apps pas WHERE p.id = pas.project_id) 'app_count',
(SELECT GROUP_CONCAT(pas.application_id) FROM  devops_projects_apps pas WHERE p.id = pas.project_id) 'app_ids'
FROM devops_projects p WHERE p.tenant_id= ? AND p.soft_del=0`
	var sqlParams []interface{}
	sqlParams = append(sqlParams, request.TenantID)
	if request.Name != "" {
		sql += " AND p.name like ?"
		sqlParams = append(sqlParams, "%"+request.Name+"%")
	}
	res := &[]dto.DevOpsProjectsDTO{}
	return page.StartPage(service.db, request.CurrentPage, request.PageSize).DoScan(res, sql, sqlParams...)
}

func (service *DevopsService) GetAppList(req *requests2.AppReq) (error, *page.Page) {
	res := &[]dto.ApplicationInfoDTO{}
	var sqlParams []interface{}
	sb := strings.Builder{}
	sb.WriteString(`SELECT t1.*,t1.git_type as SourceType,t1.sc_id as SCID,t2.name as language_name,t3.name as level_name FROM sgr_tenant_application AS t1 
INNER JOIN sgr_code_application_language AS t2 ON t1.language = t2.id 
INNER JOIN sgr_code_application_level AS t3 ON t1.LEVEL = t3.id 
INNER JOIN devops_projects_apps AS papp ON papp.application_id = t1.id
WHERE t1.status = 1 AND papp.project_id = ? `)
	sqlParams = append(sqlParams, req.ProjectID)
	if req.Name != "" {
		sb.WriteString(" AND t1.name like ?")
		sqlParams = append(sqlParams, "%"+req.Name+"%")
	}
	if req.Labels != "" {
		sb.WriteString(" AND t1.labels like ?")
		sqlParams = append(sqlParams, "%"+req.Labels+"%")
	}
	if req.Level != 0 {
		sb.WriteString(" AND t1.level=?")
		sqlParams = append(sqlParams, req.Level)
	}
	if req.Language != 0 {
		sb.WriteString(" AND t1.language=?")
		sqlParams = append(sqlParams, req.Language)
	}

	if req.TenantID > 0 {
		sb.WriteString(" AND t1.tenant_Id=?")
		sqlParams = append(sqlParams, req.TenantID)
	}

	return page.StartPage(service.db, req.PageIndex, req.PageSize).DoScan(res, sb.String(), sqlParams...)
}

func (service *DevopsService) GetPipelineListByProjectId(projectId uint64, tenantId uint64) ([]dto.ProjectPipelines, error) {
	sql := `SELECT ppl.*,app.name appName FROM devops_projects devp
INNER JOIN devops_projects_apps depa on depa.project_id = devp.id
INNER JOIN sgr_tenant_application_pipelines ppl on ppl.appid = depa.application_id 
INNER JOIN sgr_tenant_application app on app.id = depa.application_id
WHERE devp.id = ? AND devp.tenant_id = ? AND ppl.status=1 `
	var res []dto.ProjectPipelines
	err := service.db.Raw(sql, projectId, tenantId).Find(&res).Error
	if err != nil {
		return nil, err
	}
	return res, nil
}
