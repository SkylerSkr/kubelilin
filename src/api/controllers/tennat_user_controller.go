package controllers

import (
	"github.com/yoyofx/yoyogo/abstractions"
	fxutils "github.com/yoyofx/yoyogo/utils"
	"github.com/yoyofx/yoyogo/utils/jwt"
	"github.com/yoyofx/yoyogo/web/context"
	"github.com/yoyofx/yoyogo/web/mvc"
	"kubelilin/api/dto/requests"
	"kubelilin/domain/business/tenant"
	dbmodels "kubelilin/domain/database/models"
	"strconv"
	"time"
)

type UserController struct {
	mvc.ApiController
	Service *tenant.UserService
	config  struct {
		secretKey string
		expires   int
	}
}

func NewUserController(configuration abstractions.IConfiguration, service *tenant.UserService) *UserController {
	secretKey, _ := configuration.Get("yoyogo.application.server.jwt.secret").(string)
	expires, _ := configuration.Get("yoyogo.application.server.jwt.expires").(int)

	return &UserController{Service: service,
		config: struct {
			secretKey string
			expires   int
		}{secretKey: secretKey, expires: expires}}
}

func (user *UserController) PostLogin(ctx *context.HttpContext, loginRequest *requests.LoginRequest) mvc.ApiResult {
	if loginRequest.UserName == "" || loginRequest.Password == "" {
		ctx.Output.SetStatus(401)
		return user.Fail("no username or password")
	}
	pwd := fxutils.Md5String(loginRequest.Password)
	queryUser := user.Service.GetUserByNameAndPassword(loginRequest.UserName, pwd)

	if queryUser == nil {
		return mvc.ApiResult{Success: true, Message: "can not find user be", Data: requests.LoginResult{Status: "false"}}
	}

	exp := time.Now().Add(time.Duration(user.config.expires) * time.Second)

	claims := &requests.JwtCustomClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp.Unix(),
			Issuer:    queryUser.UserName,
		},
		Uid:      uint(queryUser.ID),
		TenantId: int64(queryUser.TenantID),
	}

	token, _ := jwt.CreateCustomToken([]byte(user.config.secretKey), claims)

	return user.OK(requests.LoginResult{Status: "ok", UserId: queryUser.ID, LoginType: loginRequest.LoginType, Authority: "admin", Token: token, Expires: exp.Unix()})
}

func (user *UserController) GetInfo(ctx *context.HttpContext) mvc.ApiResult {
	strId := ctx.Input.QueryDefault("id", "")
	userId, err := strconv.ParseInt(strId, 10, 32)
	if err != nil {
		return user.Fail(err.Error())
	}
	userInfo := user.Service.GetById(userId)
	if userInfo == nil {
		return user.Fail("fail")
	}

	return mvc.ApiResult{
		Success: userInfo != nil,
		Message: "获取用户信息",
		Data: requests.UserInfoResponse{
			Name:        userInfo.UserName,
			Avatar:      "https://gw.alipayobjects.com/zos/antfincdn/XAosXuNZyF/BiazfanxmamNRoxxVxka.png",
			Userid:      strconv.FormatUint(userInfo.ID, 10),
			Email:       userInfo.Email,
			Signature:   "",
			Title:       "",
			Group:       strconv.FormatUint(userInfo.TenantID, 10),
			Tags:        nil,
			NotifyCount: 0,
			UnreadCount: 0,
			Country:     "china",
			Access:      "-",
			Address:     "-",
			Phone:       userInfo.Mobile,
		},
	}
}

func (user *UserController) PostRegister(ctx *context.HttpContext) mvc.ApiResult {
	var registerUser *dbmodels.SgrTenantUser
	_ = ctx.Bind(&registerUser)

	ok := false
	retMessage := "注册成功"
	exitsUser := user.Service.GetUserByName(registerUser.UserName)
	if exitsUser == nil {
		t := time.Now()
		registerUser.Status = 1
		registerUser.CreationTime = &t
		registerUser.UpdateTime = &t
		registerUser.Password = fxutils.Md5String(registerUser.Password)
		ok = user.Service.Register(registerUser)
	} else {
		retMessage = "注册失败"
	}
	return mvc.ApiResult{
		Success: ok,
		Message: retMessage,
		Data:    registerUser,
	}
}

func (user *UserController) PostUpdate(ctx *context.HttpContext) mvc.ApiResult {
	var modifyUser *dbmodels.SgrTenantUser
	_ = ctx.Bind(&modifyUser)
	t := time.Now()
	modifyUser.UpdateTime = &t
	exitsUser := user.Service.GetUserByName(modifyUser.UserName)
	if exitsUser != nil {
		if modifyUser.Password != exitsUser.Password {
			modifyUser.Password = fxutils.Md5String(modifyUser.Password)
		}
	}

	ok := user.Service.Update(modifyUser)
	return mvc.ApiResult{
		Success: ok,
		Message: "修改成功",
	}
}

func (user *UserController) DeleteUnRegister(ctx *context.HttpContext) mvc.ApiResult {
	idStr := ctx.Input.QueryDefault("id", "")
	userId, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		panic(err)
	}
	ok := user.Service.Delete(userId)

	return mvc.ApiResult{
		Success: ok,
		Message: "删除成功",
	}
}

func (user *UserController) PutStatus(ctx *context.HttpContext) mvc.ApiResult {
	idStr := ctx.Input.QueryDefault("id", "")
	statusStr := ctx.Input.QueryDefault("status", "")
	userId, _ := strconv.ParseInt(idStr, 10, 32)
	status, _ := strconv.Atoi(statusStr)

	ok := user.Service.SetStatus(userId, status)
	return mvc.ApiResult{
		Success: ok,
	}
}

func (user *UserController) GetList(ctx *context.HttpContext) mvc.ApiResult {
	request := &requests.QueryUserRequest{}
	err := ctx.BindWithUri(request)
	if err != nil {
		panic(err)
	}
	res := user.Service.QueryUserList(request)
	return mvc.ApiResult{
		Success: res != nil,
		Data:    res,
		Message: "查询成功",
	}
}
