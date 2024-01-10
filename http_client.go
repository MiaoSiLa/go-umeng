package umeng

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	Host                   string = "http://msg.umeng.com"
	UploadPath             string = "/upload"
	StatusPath             string = "/api/status"
	CancelPath             string = "/api/cancel"
	PostPath               string = "/api/send"
	AndroidAppKey          string
	IOSAppKey              string
	AndroidAppMasterSecret string
	IOSAppMasterSecret     string
)

type Platform int32

const (
	AppAndroid Platform = 1
	AppIOS     Platform = 2
)

// 友盟 API 文档：https://developer.umeng.com/docs/67966/detail/68343
/*
可选，Android 的发送策略
"policy":{    // 可选，发送策略
    "start_time":"xx",    // 可选，定时发送时，若不填写表示立即发送
                            // 定时发送时间不能小于当前时间
                            // 格式:"yyyy-MM-dd HH:mm:ss"
                            // 注意，start_time只对任务类消息生效
    "expire_time":"xx",    // 可选，消息过期时间，其值不可小于发送时间或者start_time(如果填写了的话)
                              // 如果不填写此参数，默认为3天后过期。格式同start_time
    "max_send_num": xx,    // 可选，发送限速，每秒发送的最大条数。最小值1000
                                 //开发者发送的消息如果有请求自己服务器的资源，可以考虑此参数
    "out_biz_no":"xx"    // 可选，消息发送接口对任务类消息的幂等性保证
                            // 强烈建议开发者在发送任务类消息时填写这个字段，友盟服务端会根据这个字段对消息做去重避免重复发送
                            // 同一个appkey下面的多个消息会根据out_biz_no去重，不同发送任务的out_biz_no需要保证不同，否则会出现后发消息被去重过滤的情况
                            // 注意，out_biz_no只对任务类消息有效
}
*/
/*
可选，iOS 的发送策略
"policy":{    // 可选，发送策略
    "start_time":"xx",    // 可选，定时发送时间，若不填写表示立即发送
                                // 定时发送时间不能小于当前时间
                                // 格式: "yyyy-MM-dd HH:mm:ss"
                                // 注意，start_time只对任务生效
    "expire_time":"xx",    // 可选，消息过期时间，其值不可小于发送时间或者
                                  // start_time(如果填写了的话)
                                  // 如果不填写此参数，默认为3天后过期。格式同start_time
    "out_biz_no":"xx",    // 可选，消息发送接口对任务类消息的幂等性保证
                                 // 强烈建议开发者在发送任务类消息时填写这个字段，友盟服务端会根据这个字段对消息做去重避免重复发送
                                 // 同一个appkey下面的多个消息会根据out_biz_no去重，不同发送任务的out_biz_no需要保证不同，否则会出现后发消息被去重过滤的情况
                                 // 注意，out_biz_no只对任务类消息有效
    "apns_collapse_id":"xx"    // 可选，多条带有相同apns_collapse_id的消息，iOS设备仅展示
                                        // 最新的一条，字段长度不得超过64bytes
}
*/
type Policy struct {
	StartTime      string `json:"start_time,omitempty"`
	ExpireTime     string `json:"expire_time,omitempty"`
	MaxSendNum     int64  `json:"max_send_num,omitempty"` // Android 使用
	OutBizNo       string `json:"out_biz_no,omitempty"`
	ApnsCollapseId string `json:"apns_collapse_id,omitempty"` // iOS 使用
}

/*
必填 Android 的消息内容(最大为1840B), 包含参数说明如下(JSON格式)
"body":{    // 必填，消息体
               // 当display_type=message时，body的内容只需填写custom字段
               // 当display_type=notification时，body包含如下参数:
    "title":"xx",    // 必填，通知标题
    "text":"xx",    // 必填，通知文字描述

    // 自定义通知图标:
    "icon":"xx",    // 可选，状态栏图标ID，R.drawable.[smallIcon]，
                        // 如果没有，默认使用应用图标
                        // 图片要求为24*24dp的图标，或24*24px放在drawable-mdpi下
                        // 注意四周各留1个dp的空白像素
    "largeIcon":"xx",    // 可选，通知栏拉开后左侧图标ID，R.drawable.[largeIcon]
                              // 图片要求为64*64dp的图标
                              // 可设计一张64*64px放在drawable-mdpi下
                              // 注意图片四周留空，不至于显示太拥挤
    "img":"xx",    // 可选，通知栏大图标的URL链接。该字段的优先级大于largeIcon
                       // 厂商通道消息，目前只支持华为，链接需要以https开头不符合此要求则通过华为通道下发时不展示该图标。[华为推送](https://developer.huawei.com/consumer/cn/doc/development/HMSCore-References-V5/https-send-api-0000001050986197-V5#ZH-CN_TOPIC_0000001050986197__section165641411103315 "华为推送")搜索“图片”即可找到关于该参数的说明
                       // 该字段要求以http或者https开头，图片建议不大于100KB。
    "expand_image":"xx",    // 消息下方展示大图，支持自有通道消息展示
                                      // 厂商通道展示大图目前仅支持小米,要求图片为固定876*324px,仅处理在友盟推送后台上传的图片。如果上传的图片不符合小米的要求，则通过小米通道下发的消息不展示该图片，其他要求请参考小米推送文档[小米富媒体推送](https://dev.mi.com/console/doc/detail?pId=1278#_3_3 "小米富媒体推送")

    // 自定义通知声音:
    "sound":"xx",    // 可选，通知声音，R.raw.[sound]
                          // 如果该字段为空，采用SDK默认的声音，即res/raw/下的
                          // umeng_push_notification_default_sound声音文件。如果SDK默认声音文件不存在，则使用系统默认Notification提示音

    // 自定义通知样式:
    "builder_id": xx,    // 可选，默认为0，用于标识该通知采用的样式。使用该参数时
                              // 开发者必须在SDK里面实现自定义通知栏样式

    // 通知到达设备后的提醒方式(注意，"true/false"为字符串):
    "play_vibrate":"true/false",    // 可选，收到通知是否震动，默认为"true"
    "play_lights":"true/false",    // 可选，收到通知是否闪灯，默认为"true"
    "play_sound":"true/false",    // 可选，收到通知是否发出声音，默认为"true"

    //点击"通知"的后续行为(默认为打开app):
    "after_open":"xx",    // 可选，默认为"go_app"，值可以为:
                                 // "go_app":打开应用
                                 // "go_url":跳转到URL
                                 // "go_activity":打开特定的activity
                                 // "go_custom":用户自定义内容
    "url":"xx",    // 当after_open=go_url时，必填
                     // 通知栏点击后跳转的URL，要求以http或者https开头
    "activity":"xx",    //当after_open=go_activity时，必填。
                            // 通知栏点击后打开的Activity
    "custom":"xx"/{},    // 当display_type=message时,必填
                               // 当display_type=notification且after_open=go_custom时，必填
                               // 用户自定义内容，可以为字符串或者JSON格式。
}
*/
type AndroidBody struct {
	DisplayType string      `json:"-"`
	Title       string      `json:"title,omitempty"`
	Text        string      `json:"text,omitempty"`
	Icon        string      `json:"icon,omitempty"`
	LargeIcon   string      `json:"largeIcon,omitempty"`
	Img         string      `json:"img,omitempty"`
	ExpandImage string      `json:"expand_image,omitempty"`
	Sound       string      `json:"sound,omitempty"`
	BuilderId   int64       `json:"builder_id,omitempty"`
	PlayVibrate string      `json:"play_vibrate,omitempty"`
	PlayLights  string      `json:"play_lights,omitempty"`
	PlaySound   string      `json:"play_sound,omitempty"`
	AfterOpen   string      `json:"after_open,omitempty"`
	Url         string      `json:"url,omitempty"`
	Activity    string      `json:"activity,omitempty"`
	Custom      interface{} `json:"custom,omitempty"` // 用户自定义内容，可以为字符串或者 JSON 格式
}

/*
必填 Android 的 payload
"payload":{    // 必填，JSON格式，具体消息内容( Android 最大为 1840B )
    "display_type":"xx",    // 必填，消息类型: notification(通知)、message(消息)
    "body":{}, // 必填，消息体
    extra:{}   // 可选，JSON格式，用户自定义key-value。
}
*/
type AndroidPayload struct {
	DisplayType string            `json:"display_type,omitempty"`
	Body        AndroidBody       `json:"body,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

/*
 必填 iOS 的消息内容(最大为2012B)，包含参数说明如下(JSON格式):
"payload":{    // 必填，JSON格式，具体消息内容(iOS最大为2012B)
    "aps":{},    // 必填 严格按照APNs定义来填写
    "key1":"value1",    // 可选，用户自定义内容, "d","p"为友盟保留字段,key不可以是"d","p"
    "key2":"value2",
...
}
*/
type IOSPayload map[string]interface{}

/*
 可选 iOS 的 aps 中的 alert 部分，alert 可为如下的结构体的数据结构
"alert":""/{,    // 当content-available=1时(静默推送)，可选; 否则必填
                    // 可为字典类型和字符串类型
      "title":"title",
      "subtitle":"subtitle",
      "body":"body"
}
*/
type Alert struct {
	Title    string `json:"title,omitempty"`
	SubTitle string `json:"subtitle,omitempty"`
	Body     string `json:"body,omitempty"`
}

/*
 必填 iOS 的 payload 中的 aps 部分，消息内容
"aps":{    // 必填，严格按照APNs定义来填写
    "alert":""/{,    // 当content-available=1时(静默推送)，可选; 否则必填
                        // 可为字典类型和字符串类型
          "title":"title",
          "subtitle":"subtitle",
          "body":"body"
     }
    "badge": xx,    // 可选
    "sound":"xx",    // 可选
    "content-available":1    // 可选，代表静默推送
    "category":"xx",    // 可选，注意: ios8才支持该字段
}
*/
type IOSAps struct {
	Alert            interface{} `json:"alert,omitempty"` // 可为字典类型和字符串类型
	Badge            int64       `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int64       `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
	Image            string      `json:"image,omitempty"`           // 给 iOS 推送的图片，参考文档 https://developer.umeng.com/docs/67966/detail/66734 里的 API 发送部分
	MutableContent   int64       `json:"mutable-content,omitempty"` // 使用富文本推送，推送图片时该字段需要传 1
}

/*
调用参数-Android
{
    "appkey":"xx",    // 必填，应用唯一标识
    "timestamp":"xx",    // 必填，时间戳，10位或者13位均可，时间戳有效期为10分钟
    "type":"xx",    // 必填，消息发送类型,其值可以为:
                        // unicast-单播
                        // listcast-列播，要求不超过500个device_token
                        // filecast-文件播，多个device_token可通过文件形式批量发送
                        // broadcast-广播
                        // groupcast-组播，按照filter筛选用户群,请参照filter参数
                        // customizedcast，通过alias进行推送，包括以下两种case:
                        // -alias:对单个或者多个alias进行推送
                        // -file_id:将alias存放到文件后，根据file_id来推送
    "device_tokens":"xx",    // 当type=unicast时,必填,表示指定的单个设备
                                     // 当type=listcast时,必填,要求不超过500个,以英文逗号分隔
    "alias_type":"xx",    // 当type=customizedcast时,必填
                                // alias的类型, alias_type可由开发者自定义,开发者在SDK中调用setAlias(alias, alias_type)时所设置的alias_type
    "alias":"xx",    // 当type=customizedcast时,选填(此参数和file_id二选一)
                        // 开发者填写自己的alias,要求不超过500个alias,多个alias以英文逗号间隔
                        // 在SDK中调用setAlias(alias, alias_type)时所设置的alias
    "file_id":"xx",    // 当type=filecast时，必填，file内容为多条device_token，以回车符分割
                          // 当type=customizedcast时，选填(此参数和alias二选一)
                          // file内容为多条alias，以回车符分隔。注意同一个文件内的alias所对应的alias_type必须和接口参数alias_type一致
                          // 使用文件播需要先调用文件上传接口获取file_id，参照"文件上传"
    "filter":{},    // 当type=groupcast时，必填，用户筛选条件，如用户标签、渠道等，参考附录G
                     // filter的内容长度最大为3000B
    "payload":{},   // 必填，JSON格式，具体消息内容(Android最大为1840B)
    "policy":{},  // 可选，发送策略
    "production_mode":"true/false",    // 可选，true正式模式，false测试模式。默认为true
                                                     // 测试模式只对“广播”、“组播”类消息生效，其他类型的消息任务（如“文件播”）不会走测试模式
                                                     // 测试模式只会将消息发给测试设备。测试设备需要到web上添加
                                                     // Android:测试设备属于正式设备的一个子集
    "description":"xx",    // 可选，发送消息描述，建议填写

    "channel_properties":{}   // 可选，厂商通道相关的特殊配置
}
*/
/*
调用参数-iOS
{
    "appkey":"xx",    // 必填，应用唯一标识
    "timestamp":"xx",    // 必填，时间戳，10位或者13位均可，时间戳有效期为10分钟
    "type":"xx",    // 必填，消息发送类型,其值可以为:
                        // unicast-单播
                        // listcast-列播，要求不超过500个device_token
                        // filecast-文件播，多个device_token可通过文件形式批量发送
                        // broadcast-广播
                        // groupcast-组播，按照filter筛选用户群, 请参照filter参数
                        // customizedcast，通过alias进行推送，包括以下两种case:
                        // -alias: 对单个或者多个alias进行推送
                        // -file_id: 将alias存放到文件后，根据file_id来推送
    "device_tokens":"xx",    // 当type=unicast时, 必填, 表示指定的单个设备
                                      // 当type=listcast时, 必填, 要求不超过500个, 以英文逗号分隔
    "alias_type":"xx",    // 当type=customizedcast时, 必填
                                // alias的类型, alias_type可由开发者自定义, 开发者在SDK中调用setAlias(alias, alias_type)时所设置的alias_type
    "alias":"xx",    // 当type=customizedcast时, 选填(此参数和file_id二选一)
                        // 开发者填写自己的alias, 要求不超过500个alias, 多个alias以英文逗号间隔
                        // 在SDK中调用setAlias(alias, alias_type)时所设置的alias
    "file_id":"xx",    // 当type=filecast时，必填，file内容为多条device_token，以回车符分割
                          // 当type=customizedcast时，选填(此参数和alias二选一)
                          // file内容为多条alias，以回车符分隔。注意同一个文件内的alias所对应的alias_type必须和接口参数alias_type一致。
                          // 使用文件播需要先调用文件上传接口获取file_id，参照"2.4文件上传接口"
    "filter":{},    // 当type=groupcast时，必填，用户筛选条件，如用户标签、渠道等，参考附录G
    "payload":{},    // 必填，JSON格式，具体消息内容(iOS最大为2012B)
    "policy":{},    // 可选，发送策略
    "production_mode":"true/false",    // 可选，正式/测试模式。默认为true
                                                    // 测试模式只对“广播”、“组播”类消息生效，其他类型的消息任务（如“文件播”）不会走测试模式
                                                    // 测试模式只会将消息发给测试设备。测试设备需要到web上添加
    "description":"xx"    // 可选，发送消息描述，建议填写接口
}
*/
type Data struct {
	Platform          Platform          `json:"-"`
	AppKey            string            `json:"appkey,omitempty"`
	TimeStamp         string            `json:"timestamp,omitempty"`
	Type              string            `json:"type,omitempty"`
	DeviceTokens      string            `json:"device_tokens,omitempty"`
	FileContent       string            `json:"content,omitempty"`
	AliasType         string            `json:"alias_type,omitempty"`
	Alias             string            `json:"alias,omitempty"`
	FileId            string            `json:"file_id,omitempty"`
	Filter            interface{}       `json:"filter,omitempty"` // 比如取值为 {}
	Payload           interface{}       `json:"payload,omitempty"`
	Policy            Policy            `json:"policy,omitempty"`
	ProductionMode    string            `json:"production_mode,omitempty"`
	Description       string            `json:"description,omitempty"`
	ChannelProperties ChannelProperties `json:"channel_properties,omitempty"` // 只用于 Android
	dataBytes         []byte            `json:"-"`
}

/*
可选，厂商通道相关的特殊配置，只有 Android 可选该配置
"channel_properties":{               // 可选，厂商通道相关的特殊配置
    "channel_activity":"xxx",        // 系统弹窗，只有display_type=notification时有效，表示华为、小米、oppo、vivo、魅族的设备离线时走系统通道下发时打开指定页面acitivity的完整包路径。
    "xiaomi_channel_id":"",          // 小米channel_id，具体使用及限制请参考小米推送文档 https://dev.mi.com/console/doc/detail?pId=2086
    "vivo_classification":"1",       // vivo消息分类：0运营消息，1系统消息，需要到vivo申请，具体使用及限制参考[vivo消息推送分类功能说明]https://dev.vivo.com.cn/documentCenter/doc/359
    "vivo_category":"xx",            // vivo消息二级分类参数：友盟侧只进行参数透传，不做合法性校验，具体使用及限制参考[vivo消息推送分类功能说明]https://dev.vivo.com.cn/documentCenter/doc/359
    "oppo_channel_id":"xx"           // 可选， android8以上推送消息需要新建通道，否则消息无法触达用户。push sdk 6.0.5及以上创建了默认的通道:upush_default，消息提交厂商通道时默认添加该通道。如果要自定义通道名称或使用私信，请自行创建通道，推送消息时携带该参数具体可参考[oppo通知通道适配] https://open.oppomobile.com/wiki/doc#id=10289
    "huawei_channel_importance":"xx" // 可选，华为 & 荣耀消息分类 LOW：资讯营销类消息，NORMAL：服务与通讯类消息，具体使用及限制参考[华为消息发送方案]https://developer.huawei.com/consumer/cn/doc/HMSCore-Guides/message-priority-0000001181716924
    "huawei_channel_category":"xx"   // 可选，华为自分类消息类型，具体使用及限制参考[华为消息发送方案]https://developer.huawei.com/consumer/cn/doc/development/HMSCore-Guides/message-priority-0000001181716924
}
*/
type ChannelProperties struct {
	ChannelActivity         string `json:"channel_activity,omitempty"`
	XiaomiChannelId         string `json:"xiaomi_channAel_id,omitempty"`
	VivoClassification      string `json:"vivo_classification,omitempty"`
	VivoCategory            string `json:"vivo_category,omitempty"`
	OppoChannelId           string `json:"oppo_channel_id,omitempty"`
	HuaweiChannelImportance string `json:"huawei_channel_importance"`
	HuaweiChannelCategory   string `json:"huawei_channel_category"`
}

type response struct {
	Code string `json:"ret,omitempty"`
	Data Result `json:"data"`
}

// Result type can store the "data" section of umeng JSON response
type Result map[string]string

// APIError is the go-umeng API error type
type APIError struct {
	message string
}

func (e *APIError) Error() string {
	return e.message
}

func newAPIError(msg string) *APIError {
	return &APIError{"umeng: " + msg}
}

func NewData(pf Platform) (data *Data) {
	data = new(Data)
	data.Platform = pf
	if data.Platform == AppAndroid {
		data.AppKey = AndroidAppKey
	}
	if data.Platform == AppIOS {
		data.AppKey = IOSAppKey
	}
	return
}

func (data *Data) SetPolicy(policy Policy) {
	data.Policy = policy
}

func (data *Data) Push(body, aps, policy interface{}, extras map[string]string) (result Result, err error) {
	if data.Platform == AppAndroid {
		// Doc: http://dev.umeng.com/push/android/api-doc#2_1_3
		payload := &AndroidPayload{}
		if v, ok := body.(AndroidBody); ok {
			if v.DisplayType != "message" && v.DisplayType != "notification" {
				panic("invalid display_type field")
			}
			if v.DisplayType == "message" || (v.DisplayType == "notification" && v.AfterOpen == "go_custom") {
				switch c := v.Custom.(type) {
				case string:
					if len(c) == 0 {
						panic("missing custom field")
					}
				case map[string]interface{}:
					// 只能传 json 对象不可传 json 数组
					if len(c) == 0 {
						panic("missing custom field")
					}
				default:
					panic("invalid custom field")
				}
			}
			payload.DisplayType = v.DisplayType
			payload.Body = v
		}
		if len(extras) > 0 {
			payload.Extra = extras
		}
		data.Payload = payload
	} else if data.Platform == AppIOS {
		// Doc: http://dev.umeng.com/push/ios/api-doc#2_1_3
		payload := make(IOSPayload, 0)
		payload["aps"] = aps
		if len(extras) > 0 {
			for key, val := range extras {
				payload[key] = val
			}

		}
		data.Payload = payload
	}

	if v, ok := policy.(Policy); ok {
		data.SetPolicy(v)
	}
	return data.Send(data.Sign(PostPath))
}

func (data *Data) Status() (result Result, err error) {
	return data.Send(data.Sign(StatusPath))
}

func (data *Data) Cancel() (result Result, err error) {
	return data.Send(data.Sign(CancelPath))
}

func (data *Data) Upload() (result Result, err error) {
	return data.Send(data.Sign(UploadPath))
}

func (data *Data) Sign(reqPath string) (api string) {
	data.dataBytes, _ = json.Marshal(data)
	jsonStr := string(data.dataBytes)
	sign := ""
	if data.Platform == AppAndroid {
		sign = Md5(fmt.Sprintf("POST%s%s%s%s", Host, reqPath, jsonStr, AndroidAppMasterSecret))
	} else if data.Platform == AppIOS {
		sign = Md5(fmt.Sprintf("POST%s%s%s%s", Host, reqPath, jsonStr, IOSAppMasterSecret))
	}
	api = fmt.Sprintf("%s%s?sign=%s", Host, reqPath, sign)
	return
}

func (data *Data) Send(url string) (Result, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data.dataBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result response
	err = json.Unmarshal(body, &result)
	if err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, newAPIError("JSON parse error, HTTP " + resp.Status)
		}
		return nil, err
	}

	if result.Code != "SUCCESS" {
		data, er := json.Marshal(result.Data)
		if er != nil {
			err = newAPIError("unexpected response content")
		} else {
			err = newAPIError(string(data))
		}
		return nil, err
	}
	return result.Data, nil
}

func Md5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
