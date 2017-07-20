package handler

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"net/http"
	"service/common"
	"service/conf"
	"service/module"
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

	n, _ := c.Request.Body.Read(reqMsgBuf)

	err := json.Unmarshal(reqMsgBuf[:n], create_request)
	if err != nil {
		glog.Error("[SessionCreate] request json data unmarshal err = [", err, "]")
		create_response = makeCreateResponse(create_request.UserID, "", "", -10000, "request json format error")
	} else {
		glog.Info(">>> [SessionCreate] Request json data = [", create_request, "]")

		s := &module.Session{
			UserID:      create_request.UserID,
			WxWebCommon: common.DefaultCommon,
			WxWebXcg:    &conf.XmlConfig{},
			WxApi:       &wxapi.WebwxApi{},
			CreateTime:  time.Now().Unix(),
			LoginStat:   0,
			Loop:        false,
			Quit:        make(chan bool),
		}

		s.UuID, s.QRcode = s.WxApi.WebwxGetUuid(s.WxWebCommon)
		if s.UuID != "" && s.QRcode != "" {
			glog.Info(">>> [SessionCreate] UserID ", s.UserID, "'s uuid = ", s.UuID, " && qrcode = ", s.QRcode)
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
		s.AutoRepliesConf[i].GroupNickName, _ = request.Config[i]["group"].(string)

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
			glog.Info(">>> [InitSession] Group [", s.AutoRepliesConf[i].GroupNickName, "] keyword configs = [", s.AutoRepliesConf, "]")
		} else {
			glog.Info(">>> [InitSession] Group [", s.AutoRepliesConf[i].GroupNickName, "] has no keywords")
		}
	}
}
