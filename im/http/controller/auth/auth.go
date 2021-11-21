/**
  @author:panliang
  @data:2021/6/18
  @note
**/
package auth

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"

	"im_app/im/http/dao"
	userModel "im_app/im/http/models/user"
	"im_app/im/http/validates"
	"im_app/im/utils"
	"im_app/im/ws"
	"im_app/pkg/config"
	"im_app/pkg/helpler"
	"im_app/pkg/jwt"
	"im_app/pkg/model"
	"im_app/pkg/response"
)

type (
	AuthController  struct{}
	WeiBoController struct{}
	Me              struct {
		ID             uint64 `json:"id"`
		Name           string `json:"name"`
		Avatar         string `json:"avatar"`
		Email          string `json:"email"`
		Token          string `json:"token"`
		ExpirationTime int64  `json:"expiration_time"`
		Sex            int    `json:"sex"`
		ClientType     int    `json:"client_type"`
		Bio            string `json:"bio"`
	}
)

// @BasePath /api

// @Summary 获取用户信息接口
// @Description 获取用户信息接口
// @Tags 获取用户信息接口
// @SecurityDefinitions.apikey ApiKeyAuth
// @In header
// @Name Authorization
// @Param Authorization	header string true "Bearer 31a165baebe6dec616b1f8f3207b4273"
// @Produce json
// @Success 200
// @Router /me [post]
func (*AuthController) Me(c *gin.Context) {
	user := userModel.AuthUser
	response.SuccessResponse(user, 200).ToJson(c)
}

// @BasePath /api

// @Summary 更新用户数据
// @Description 更新用户数据
// @Tags 更新用户数据
// @SecurityDefinitions.apikey ApiKeyAuth
// @In header
// @Name Authorization
// @Param Authorization	header string true "Bearer 31a165baebe6dec616b1f8f3207b4273"
// @Param bio formData string false "个性签名"
// @Param six formData int false "性别"

// @Produce json
// @Success 200
// @Router /Update [put]
func (*AuthController) Update(c *gin.Context) {
	bio := c.PostForm("bio")
	sex := c.PostForm("sex")
	user := userModel.AuthUser
	var users userModel.Users
	if len(sex) != 0 {
		sex, _ := strconv.Atoi(sex)
		users.Sex = sex
	}
	users.Bio = bio

	model.DB.Table("users").Where("id", user.ID).First(&users)
	users.Bio = bio
	model.DB.Save(&users)
	response.SuccessResponse().WriteTo(c)
	return
}

// @BasePath /api

// @Summary 这是一个登录接口
// @Description 登录接口
// @Tags 登录接口
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "账号"
// @Param password formData string true "密码"
// @Param client_type formData string false "客户端类型 0.网页端登录 1.设备端登录"
// @Success 200
// @Router /login [post]
func (that *AuthController) Login(c *gin.Context) {
	_user := userModel.Users{
		Name:     c.PostForm("name"),
		Password: c.PostForm("password"),
	}

	ClientType, _ := strconv.Atoi(c.DefaultPostForm("client_type", "0"))

	errs := validates.ValidateLoginForm(_user)
	if len(errs) > 0 {
		response.ErrorResponse(500, "参数错误", errs).WriteTo(c)
		return
	}
	var users userModel.Users
	model.DB.Model(&userModel.Users{}).Where("name = ?", _user.Name).Find(&users)
	if users.ID == 0 {
		response.FailResponse(403, "用户不存在").ToJson(c)
		return
	}
	if !helpler.ComparePasswords(users.Password, _user.Password) {
		response.FailResponse(403, "账号或者密码错误").ToJson(c)
		return
	}

	users.ClientType = ClientType
	model.DB.Model(&userModel.Users{}).Save(&users)
	// 挤下线操作
	if users.Status == 1 {
		ws.CrowdedOffline(strconv.Itoa(int(users.ID)))
	}
	token := jwt.GenerateToken(users.ID, users.Name, users.Avatar, users.Email, ClientType)
	data := getMe(token, &users)
	response.SuccessResponse(data, 200).ToJson(c)
}

func (*WeiBoController) WeiBoCallBack(c *gin.Context) {
	code := c.Query("code")
	if len(code) == 0 {
		response.FailResponse(403, "参数不正确~").ToJson(c)
	}
	access_token := utils.GetWeiBoAccessToken(&code)
	UserInfo := utils.GetWeiBoUserInfo(&access_token)
	users := userModel.Users{}
	oauth_id := gjson.Get(UserInfo, "id").Raw
	isThere := model.DB.Where("oauth_id = ?", oauth_id).First(&users)
	if isThere.Error != nil {
		userData := userModel.Users{
			Email:           gjson.Get(UserInfo, "email").Str,
			Password:        helpler.HashAndSalt("123456"),
			PasswordComfirm: helpler.HashAndSalt("123456"),
			OauthId:         gjson.Get(UserInfo, "id").Raw,
			Avatar:          gjson.Get(UserInfo, "avatar_large").Str,
			Name:            gjson.Get(UserInfo, "name").Str,
			OauthType:       1,
			CreatedAt:       time.Unix(time.Now().Unix(), 0).Format("2006-01-02 15:04:05"),
		}
		result := model.DB.Create(&userData)

		if result.Error != nil {
			response.FailResponse(500, "用户微博授权失败").ToJson(c)
		} else {
			// 执行默认添加好友逻辑
			dao := new(dao.UserService)
			dao.AddDefaultFriend(users.ID)
			token := jwt.GenerateToken(users.ID, users.Name, users.Avatar, users.Email, 0)
			data := getMe(token, &users)
			response.SuccessResponse(data, 200).ToJson(c)
		}
	} else {
		token := jwt.GenerateToken(users.ID, users.Name, users.Avatar, users.Email, 0)
		data := getMe(token, &users)
		response.SuccessResponse(data, 200).ToJson(c)

	}
}

func (*AuthController) WxCallback(c *gin.Context) {
	response.SuccessResponse().ToJson(c)
	return
}

func getMe(token string, user *userModel.Users) *Me {
	expiration_time := config.GetInt64("app.jwt.expiration_time")
	data := new(Me)
	data.ID = user.ID
	data.Name = user.Name
	data.Avatar = user.Avatar
	data.Email = user.Email
	data.Token = token
	data.ExpirationTime = expiration_time
	data.Bio = user.Bio
	data.Sex = user.Sex
	data.ClientType = user.ClientType
	return data
}
