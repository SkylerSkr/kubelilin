package app

import (
	"encoding/json"
	"errors"
	"github.com/jinzhu/copier"
	"gorm.io/gorm"
	"kubelilin/api/dto/requests"
	"kubelilin/domain/database/models"
	"kubelilin/domain/dto"
	"kubelilin/pkg/page"
	"strconv"
	"strings"
)

const (
	Deployment  = "Deployment"
	DaemonSet   = "DaemonSet"
	StatefulSet = "StatefulSet"
	CronJob     = "CronJob"
)

type DeploymentService struct {
	db *gorm.DB
}

func NewDeploymentService(db *gorm.DB) *DeploymentService {
	return &DeploymentService{db: db}
}

func (deployment *DeploymentService) NewOrUpdateDeployment(deployModel *requests.DeploymentStepRequest) (error, *requests.DeploymentStepRequest) {
	var deploy *models.SgrTenantDeployments

	deployment.db.Model(deployModel).Where("app_id = ? and name = ?", deployModel.AppID, deployModel.Name).First(&deploy)
	if deploy != nil { // new
		dbRes := deployment.db.Model(deployModel).Create(deployModel)
		return dbRes.Error, deployModel
	} else { // update
		dbRes := deployment.db.Model(deployModel).Updates(deployModel)
		return dbRes.Error, deployModel
	}
}

func (deployment *DeploymentService) CreateDeploymentStep1(deployModel *requests.DeploymentStepRequest) (error, *models.SgrTenantDeployments) {
	dpModel := &models.SgrTenantDeployments{}
	dpModel.ServiceName = deployModel.Name + "-svc"
	dpModel.WorkloadType = "Deployment"
	dpModel.ServicePortType = "TCP"
	err := copier.Copy(dpModel, deployModel)
	if err != nil {
		return err, nil
	}
	svcPort, _ := strconv.ParseUint(deployModel.ServicePort, 10, 32)

	dpModel.ServicePort = uint(svcPort)
	//名称端口重复性校验
	if deployModel.ID > 0 {
		//var existCount int64
		//deployment.db.Model(&models.SgrTenantDeployments{}).Where("service_away=? and service_port=? and id !=?", deployModel.ServiceAway, deployModel.ServicePort, deployModel.ID).Count(&existCount)
		//if existCount > 0 {
		//	return errors.New("已经存在相同的服务端口"), nil
		//}
		dbRes := deployment.db.Model(&models.SgrTenantDeployments{}).Where("id=?", deployModel.ID).Updates(map[string]interface{}{models.SgrTenantDeploymentsColumns.Nickname: deployModel.Nickname,
			models.SgrTenantDeploymentsColumns.ServiceEnable: deployModel.ServiceEnable, models.SgrTenantDeploymentsColumns.ServicePort: deployModel.ServicePort})
		return dbRes.Error, dpModel
	} else {
		var existCount int64
		deployment.db.Model(&models.SgrTenantDeployments{}).Where("name=?", deployModel.Name).Count(&existCount)
		if existCount > 0 {
			return errors.New("已经存在相同的部署"), nil
		}
		//deployment.db.Model(&models.SgrTenantDeployments{}).Where("service_away=? and service_port=? and id!=?", deployModel.ServiceAway, deployModel.ServicePort, deployModel.ID).Count(&existCount)
		//if existCount > 0 {
		//	return errors.New("已经存在相同的服务端口"), nil
		//}
		dbRes := deployment.db.Model(&models.SgrTenantDeployments{}).Create(dpModel)
		return dbRes.Error, dpModel
	}
}

func (deployment *DeploymentService) CreateDeploymentStep2(deployModel *requests.DeploymentStepRequest) (error, *models.SgrTenantDeployments) {
	dpModel := models.SgrTenantDeployments{}
	/*requestCPU, _ := strconv.ParseFloat(deployModel.RequestCPU, 64)
	requestMemory, _ := strconv.ParseFloat(deployModel.RequestMemory, 64)
	limitCPU, _ := strconv.ParseFloat(deployModel.LimitCPU, 64)
	limitMemory, _ := strconv.ParseFloat(deployModel.LimitMemory, 64)*/

	envJson, jsonErr := json.Marshal(deployModel.Environments)
	if jsonErr != nil {
		return jsonErr, nil
	}
	dpcModel := models.SgrTenantDeploymentsContainers{
		DeployID:      deployModel.ID,
		IsMain:        1,
		RequestCPU:    deployModel.RequestCPU,
		RequestMemory: deployModel.RequestMemory,
		LimitCPU:      deployModel.LimitCPU,
		LimitMemory:   deployModel.LimitMemory,
		Environments:  string(envJson),
	}
	deployment.db.Model(&models.SgrTenantDeployments{}).Where("id = ?", deployModel.ID).First(&dpModel)
	if dpModel.AppID == 0 {
		return errors.New("未找到相应的部署数据"), nil
	}
	tsRes := deployment.db.Transaction(func(tx *gorm.DB) error {
		//更新副本数
		dpRes := tx.Model(&models.SgrTenantDeployments{}).Where("id = ?", deployModel.ID).Update(models.SgrTenantDeploymentsColumns.Replicas, deployModel.Replicas)
		if dpRes.Error != nil {
			return dpRes.Error
		}
		var existDpc models.SgrTenantDeploymentsContainers
		tx.Model(&models.SgrTenantDeploymentsContainers{}).Where("deploy_id=?", deployModel.ID).First(&existDpc)
		if existDpc.ID > 0 {
			//更新CPU 内存限制
			dpcRes := tx.Model(&models.SgrTenantDeploymentsContainers{}).Where("id=?", existDpc.ID).Updates(&dpcModel)
			if dpcRes.Error != nil {
				return dpcRes.Error
			}
		} else {
			//创建CPU 内存限制
			dpcRes := tx.Model(&models.SgrTenantDeploymentsContainers{}).Create(&dpcModel)
			if dpcRes.Error != nil {
				return dpcRes.Error
			}
		}

		return nil
	})
	if tsRes != nil {
		return tsRes, nil
	}
	return nil, &dpModel
}

func (deployment *DeploymentService) GetDeployments(profile string, appId uint64, tenantId uint64,
	deployName string, appName string, clusterId uint64, projectId uint64, pageIndex int, pageSize int) (error, *page.Page) {

	dataSql := strings.Builder{}
	dataSql.WriteString(`SELECT d.id, d.nickname ,d.name,lev.name level, c.name  as 'clusterName' ,app.name as 'appName',
  d.cluster_id as 'clusterId' , n.namespace ,d.last_image as 'lastImage', 0 'running' , 
  d.replicas 'expected', '0.0.0.0' as 'serviceIP', d.service_name as 'serviceName',d.service_Port as 'servicePort'
  FROM sgr_tenant_deployments d
  INNER JOIN sgr_tenant_cluster c on c.id = d.cluster_id
  INNER JOIN sgr_tenant_namespace n on n.id = d.namespace_id
  INNER JOIN sgr_tenant_application app on  app.id = d.app_id
  INNER JOIN sgr_code_deployment_level lev on lev.code = d.level
  WHERE d.tenant_id =? `)

	if deployName != "" {
		dataSql.WriteString("AND d.name like '%" + deployName + "%'")
	}

	if appName != "" {
		dataSql.WriteString("AND app.name like '%" + appName + "%'")
	}
	var params []interface{}
	params = append(params, tenantId)

	if clusterId > 0 {
		dataSql.WriteString(" AND c.id = ? ")
		params = append(params, clusterId)
	}
	if profile != "" {
		dataSql.WriteString("AND lev.code = ?")
		params = append(params, profile)
	}

	if projectId > 0 {
		dataSql.WriteString(" AND app.id in (select application_id from devops_projects_apps WHERE project_id =?)")
		params = append(params, projectId)
	} else {
		if appId > 0 {
			dataSql.WriteString(" AND d.app_id = ? ")
			params = append(params, appId)
		}
	}

	var deploymentList []dto.DeploymentItemDto
	return page.StartPage(deployment.db, pageIndex, pageSize).DoScan(&deploymentList, dataSql.String(), params...)
	//dataRes := deployment.db.Raw(dataSql.String(), params...).Scan(&deploymentList)
	//return deploymentList, dataRes.Error
}

func (deployment *DeploymentService) GetDeploymentForm(id uint64) (error, *requests.DeploymentStepRequest) {

	res := &requests.DeploymentStepRequest{}
	sql := strings.Builder{}
	sql.WriteString(`select dp.id,dpc.id as dpc_id , dp.name,dp.nickname,dp.tenant_id,dp.cluster_id,dp.namespace_id,dp.app_id,dp.app_name,
       dp.level,dp.replicas,dp.service_away,dp.service_enable,dp.service_port,dp.service_port_type,
       dpc.request_cpu,dpc.limit_cpu,dpc.request_memory,dpc.limit_memory,dpc.environments as env_json
       from sgr_tenant_deployments as dp
inner join sgr_tenant_deployments_containers as dpc on dp.id=dpc.deploy_id
where dp.id=?`)
	resErr := deployment.db.Raw(sql.String(), id).Scan(res)
	if resErr != nil && res.EnvJson != "" {
		var env []requests.DeploymentEnv
		jsonErr := json.Unmarshal([]byte(res.EnvJson), &env)
		if jsonErr != nil {
			return jsonErr, nil
		}
		res.Environments = env
	}
	return resErr.Error, res
}

func (deployment *DeploymentService) GetDeploymentByID(id uint64) (dto.DeploymentItemDto, error) {
	var dpModel dto.DeploymentItemDto
	sql := strings.Builder{}
	sql.WriteString(`SELECT d.id, d.nickname ,d.name,
  d.cluster_id as 'clusterId' , n.namespace ,d.replicas 'expected',  d.service_name as 'serviceName', d.service_port as 'servicePort'
  FROM sgr_tenant_deployments d
  INNER JOIN sgr_tenant_namespace n on n.id = d.namespace_id
  WHERE  d.id = ?`)
	resErr := deployment.db.Raw(sql.String(), id).Scan(&dpModel)
	return dpModel, resErr.Error
}

func (deployment *DeploymentService) SetReplicas(id uint64, number int32) error {
	dpRes := deployment.db.Model(&models.SgrTenantDeployments{}).Where("id = ?", id).Update(models.SgrTenantDeploymentsColumns.Replicas, number)
	return dpRes.Error
}
