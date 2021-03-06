package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"io"
	"net/http"
	"service/common"
	"service/conf"
	"service/module"
	"service/utils"
	"service/wxapi"
	"time"
)

func makeCreateResponse(uid int, uuid, qrcode string, code int, msg string) *common.Msg_Create_Response {
	resp := &common.Msg_Create_Response{
		Action: conf.CLIENT_CREATE,
		UserID: uid,
		Code:   code,
		Msg:    msg,
		Uuid:   uuid,
		QrCode: qrcode,
	}
	return resp
}

func SessionCreate(c *gin.Context) {
	create_request := &common.Msg_Create_Request{}
	create_response := &common.Msg_Create_Response{}

	reqMsgBuf := make([]byte, conf.MAX_BUF_SIZE)
	//n, _ := c.Request.Body.Read(reqMsgBuf)		--error
	n, _ := io.ReadFull(c.Request.Body, reqMsgBuf)

	err := json.Unmarshal(reqMsgBuf[:n], create_request)
	if err != nil {
		glog.Error("[SessionCreate] request json data unmarshal err = [", err, "]")
		create_response = makeCreateResponse(create_request.UserID, "", "", -10000, "request json format error")
	} else {
		glog.Info(">>> [SessionCreate] Request JSON Data = [", create_request, "]")

		s := &module.Session{
			UserID: create_request.UserID,
			WxWebCommon: &common.Common{
				AppId:      common.DefaultCommon.AppId,
				Fun:        common.DefaultCommon.Fun,
				LoginUrl:   common.DefaultCommon.LoginUrl,
				Lang:       common.DefaultCommon.Lang,
				DeviceID:   "e" + utils.GetRandomStringFromNum("0123456789", 15),
				UserAgent:  common.DefaultCommon.UserAgent,
				MediaCount: common.DefaultCommon.MediaCount,
			},
			WxWebXcg:    &conf.XmlConfig{},
			WxApi:       &wxapi.WebwxApi{},
			CreateTime:  time.Now().Unix(),
			LoginStatus: 0,
			Loop:        false,
			MediaID:	 "",
		}

		if conf.Config.DB_ON {
			s.DBSession = module.GetDBInstant()
		}

		s.UuID, s.QRcode = s.WxApi.WebwxGetUuid(s.WxWebCommon)
		if s.UuID != "" && s.QRcode != "" {
			glog.Info(">>> [SessionCreate] UserID ", s.UserID, "'s uuid = ", s.UuID, " && qrcode = ", s.QRcode)

			defer func() {
				if err := recover(); err != nil {
					glog.Error("[SessionCreate] Recover error = [", err, "]")
				}
			}()

			InitSession(s, create_request)
			create_response = makeCreateResponse(s.UserID, s.UuID, s.QRcode, 200, "success")
		} else {
			create_response = makeCreateResponse(s.UserID, s.UuID, s.QRcode, -10001, "get uuid from wechat failed")
		}
	}

	c.JSON(http.StatusOK, create_response)
	return
}

func InitSession(s *module.Session, request *common.Msg_Create_Request) {
	if old, exist := module.SessionTable[s.UserID]; exist {
		if old.Loop {
			old.Stop()
			delete(module.SessionTable, s.UserID)
			glog.Info(">>> [InitSession] Delete UserID ", s.UserID, "'s session & loop")
		} else {
			delete(module.SessionTable, s.UserID)
			glog.Info(">>> [InitSession] Delete UserID ", s.UserID, "'s session")
		}
	}

	module.SessionTable[s.UserID] = s
	s.AutoRepliesConf = make([]module.AutoReplyConf, len(request.Config))

	for i := 0; i < len(request.Config); i++ {
		s.AutoRepliesConf[i].NickName, _ = request.Config[i]["nickname"].(string)
		s.AutoRepliesConf[i].UserType = int(request.Config[i]["type"].(float64))

		wlmText, exist := request.Config[i]["wlm_text"].(string)
		if exist {
			s.AutoRepliesConf[i].WlmText = wlmText
		} else {
			s.AutoRepliesConf[i].WlmText = ""
		}

		wlmImage, exist := request.Config[i]["wlm_image"].(string)
		if exist {
			s.AutoRepliesConf[i].WlmImage = wlmImage
		} else {
			s.AutoRepliesConf[i].WlmImage = ""
		}

		if s.AutoRepliesConf[i].UserType == conf.USER_PERSON {
			massText, exist := request.Config[i]["mass_text"].(string)
			if exist {
				s.AutoRepliesConf[i].MassText = massText
			} else {
				s.AutoRepliesConf[i].MassText = ""
			}

			massImage, exist := request.Config[i]["mass_image"].(string)
			if exist {
				s.AutoRepliesConf[i].MassImage = massImage
			} else {
				s.AutoRepliesConf[i].MassImage = ""
			}
		} else {
			s.AutoRepliesConf[i].MassText = ""
			s.AutoRepliesConf[i].MassImage = ""
		}

		sections, exist := request.Config[i]["keywords"].([]interface{})
		if exist {
			s.AutoRepliesConf[i].KeyWords = make([]module.KeyWord, len(sections))

			for j := 0; j < len(sections); j++ {
				section, exist := sections[j].(map[string]interface{})
				if exist {
					key, ok := section["keyword"].(string)
					if ok {
						s.AutoRepliesConf[i].KeyWords[j].Key = key
					} else {
						s.AutoRepliesConf[i].KeyWords[j].Key = ""
					}

					content, exist := section["text"].(string)
					if exist {
						s.AutoRepliesConf[i].KeyWords[j].Text = content
					} else {
						s.AutoRepliesConf[i].KeyWords[j].Text = ""
					}

					img, exist := section["image"].(string)
					if exist {
						s.AutoRepliesConf[i].KeyWords[j].Image = img
					} else {
						s.AutoRepliesConf[i].KeyWords[j].Image = ""
					}
				}
			}
			glog.Info(">>> [InitSession] NickName [", s.AutoRepliesConf[i].NickName, "] , type = [", s.AutoRepliesConf[i].UserType, "] keyword configs = [", s.AutoRepliesConf, "]")
		} else {
			glog.Info(">>> [InitSession] NickName [", s.AutoRepliesConf[i].NickName, "], type = [", s.AutoRepliesConf[i].UserType, "] has no keywords")
		}
	}
}
