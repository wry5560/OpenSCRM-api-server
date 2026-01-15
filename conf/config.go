package conf

import (
	"github.com/spf13/viper"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var Settings *config

type config struct {
	App        AppConfig
	Server     serverConfig
	Redis      redisConfig
	DB         DBConfig
	DelayQueue delayQueueConfig
	Storage    StorageConfig
	WeWork     weWorkConfig
	MingDaoYun MingDaoYunConfig
}

// MingDaoYunConfig 明道云配置
type MingDaoYunConfig struct {
	// APIBase 明道云 API 基础地址
	APIBase string `json:"api_base"`
	// AppKey 明道云应用 Key
	AppKey string `json:"app_key"`
	// Sign 明道云签名
	Sign string `json:"sign"`
	// CustomerWorksheetID 客户表工作表 ID
	CustomerWorksheetID string `json:"customer_worksheet_id"`
}

// delayQueueConfig  基于redis延迟队列的配置
type delayQueueConfig struct {
	// bucket数量
	BucketSize int `validate:"required"`
	// bucket在redis中的键名
	BucketName string `validate:"required"`
	// ready queue在redis中的键名
	QueueName string `validate:"required"`
	//调用blpop阻塞超时时间, 单位秒, 修改此项, redis.read_timeout必须做相应调整
	QueueBlockTimeout int `validate:"required"`
}

type AppConfig struct {
	Name               string `validate:"required"`
	Key                string `validate:"required,base64"` //应用秘钥 64位，生成命令：openssl rand 64 -base64
	Env                string `validate:"required,oneof=PROD DEV TEST DEMO"`
	AutoMigration      bool
	AutoSyncWeWorkData bool // 启动时同步微信数据
	// SuperAdmin 此处userID对应员工的赋予超级管理员权限
	SuperAdmin      []string `validate:"required,dive,gt=1"`
	InnerSrvAppCode string   // 内部服务调用key
}

type serverConfig struct {
	RunMode         string `validate:"required,oneof=debug test release"`
	HttpPort        int    `validate:"required,gt=0"`
	HttpHost        string
	ReadTimeout     time.Duration `validate:"required,gt=0"`
	WriteTimeout    time.Duration `validate:"required,gt=0"`
	MsgArchHttpPort int
	MsgArchSrvHost  string
}

// weWorkConfig 企业微信配置
type weWorkConfig struct {
	// ExtCorpID 外部企业ID
	ExtCorpID string `json:"ext_corp_id" validate:"required,corp_id"`
	// ContactSecret 通讯录secret
	ContactSecret string `json:"contact_secret" validate:"required"`
	// CustomerSecret 客户联系secret
	CustomerSecret string `json:"customer_secret" validate:"required"`
	// MainAgentID 企业主应用AgentID
	MainAgentID int64 `json:"main_agent_id" validate:"number,gt=0"`
	// MainAgentSecret 企业主应用secret
	MainAgentSecret string `json:"main_agent_secret" validate:"required"`
	// CallbackToken 企业微信事件回调Token
	CallbackToken string `json:"callback_token" validate:"required"`
	// CallbackAesKey 企业微信事件回调AesKey
	CallbackAesKey string `json:"callback_aes_key" validate:"required"`
	// PriKeyPath 会话存档解密私钥
	PriKeyPath string `json:"pri_key_path"`
	// MsgArchBatchSize 会话存档拉取，每次拉取的条数
	MsgArchBatchSize int `json:"msg_arch_batch_size"`
	// MsgArchTimeout 会话存档拉取，超时时间
	MsgArchTimeout int `json:"msg_arch_timeout"`
	// MsgArchProxy  会话存档拉取代理地址
	MsgArchProxy string `json:"msg_arch_proxy"`
	// MsgArchProxyPasswd  会话存档拉取代理密码
	MsgArchProxyPasswd string `json:"msg_arch_proxy_passwd"`
}

type DBConfig struct {
	User     string `validate:"required"`
	Password string `validate:"required"`
	Host     string `validate:"required"`
	Name     string `validate:"required"`
	SSLMode  string `validate:"omitempty,oneof=disable require verify-ca verify-full"`
}

// redisConfig redis config
type redisConfig struct {
	Host        string        `validate:"required"`
	Password    string        `validate:"required"`
	IdleTimeout time.Duration `validate:"required"`
	DBNumber    int           `validate:"gte=0"`
	DialTimeout time.Duration `validate:"required"`
	ReadTimeout time.Duration `validate:"required"`
}

type StorageConfig struct {
	// Type 存储类型, 可配置aliyun, qcloud, local；分别对应阿里云OSS, 腾讯云COS, 本地存储
	// 留空则禁用存储功能
	Type string `validate:"omitempty,oneof=aliyun qcloud local"`
	// CdnURL CDN绑定域名，可选配置，本地存储必填
	CdnURL string `validate:"omitempty,url"`

	// 阿里云OSS相关配置，请使用子账户凭据，且仅授权oss访问权限
	AccessKeyId     string `validate:"required_if=Type aliyun"`
	AccessKeySecret string `validate:"required_if=Type aliyun"`
	EndPoint        string `validate:"required_if=Type aliyun"`
	Bucket          string `validate:"required_if=Type aliyun"`

	// 腾讯云OSS相关配置，请使用子账户凭据，且仅授权cos访问权限
	SecretID  string `validate:"required_if=Type qcloud"`
	SecretKey string `validate:"required_if=Type qcloud"`
	BucketURL string `validate:"required_if=Type qcloud"`

	// 本地存储相关配置
	// LocalRootPath 本地存储文件的根目录，必须是绝对路径
	LocalRootPath string `validate:"required_if=Type local"`
	// ServerRootPath 文件服务的根目录，http服务中的文件根目录，相对路径，用于识别文件服务请求的路径标识
	ServerRootPath string `validate:"required_if=Type local"`
}

// SetupSetting Setup initialize the configuration instance
func SetupSetting() error {
	var err error
	viper.SetConfigName("config")     // name of config file (without extension)
	viper.AddConfigPath("conf")       // optionally look for config in the working directory
	viper.AddConfigPath("../conf")    // optionally look for config in the working directory
	viper.AddConfigPath("../../conf") // optionally look for config in the working directory
	viper.AddConfigPath("/srv")       // optionally look for config in the working directory
	err = viper.ReadInConfig()        // Find and read the config file
	if err != nil {                   // Handle errors reading the config file
		log.Printf("missing config.yaml : %s\n", err.Error())
		return err
	}
	Settings = &config{}
	err = viper.Unmarshal(Settings)
	if err != nil {
		log.Printf("parse config.yaml failed : %s", err.Error())
		return err
	}
	viper.WatchConfig()
	Settings.Server.ReadTimeout = Settings.Server.ReadTimeout * time.Second
	Settings.Server.WriteTimeout = Settings.Server.WriteTimeout * time.Second

	Settings.Redis.DialTimeout = Settings.Redis.DialTimeout * time.Second
	Settings.Redis.IdleTimeout = Settings.Redis.IdleTimeout * time.Second
	Settings.Redis.ReadTimeout = Settings.Redis.ReadTimeout * time.Second
	return nil
}

// SetupTestSetting 初始化单元测试的配置
func SetupTestSetting() error {
	var err error
	viper.SetConfigName("config.test") // name of config file (without extension)
	viper.AddConfigPath("conf")        // optionally look for config in the working directory
	viper.AddConfigPath("../conf")     // optionally look for config in the working directory
	viper.AddConfigPath("../../conf")  // optionally look for config in the working directory
	err = viper.ReadInConfig()         // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		log.Printf("missing config.yaml : %s\n", err.Error())
		return err
	}
	Settings = &config{}
	err = viper.Unmarshal(Settings)
	if err != nil {
		log.Printf("parse config.yaml failed : %s", err.Error())
		return err
	}
	viper.WatchConfig()
	Settings.Server.ReadTimeout = Settings.Server.ReadTimeout * time.Second
	Settings.Server.WriteTimeout = Settings.Server.WriteTimeout * time.Second

	Settings.Redis.DialTimeout = Settings.Redis.DialTimeout * time.Second
	Settings.Redis.IdleTimeout = Settings.Redis.IdleTimeout * time.Second
	Settings.Redis.ReadTimeout = Settings.Redis.ReadTimeout * time.Second
	return nil
}

// SetupSettingFromEnv 从环境变量加载配置
func SetupSettingFromEnv() error {
	Settings = &config{
		App: AppConfig{
			Name:               getEnv("APP_NAME", "openscrm"),
			Key:                getEnvRequired("APP_KEY"),
			Env:                getEnv("APP_ENV", "DEV"),
			AutoMigration:      getEnvBool("APP_AUTO_MIGRATION", true),
			AutoSyncWeWorkData: getEnvBool("APP_AUTO_SYNC_WEWORK", false),
			SuperAdmin:         strings.Split(getEnv("APP_SUPER_ADMIN", "admin"), ","),
			InnerSrvAppCode:    getEnv("APP_INNER_SRV_CODE", ""),
		},
		Server: serverConfig{
			RunMode:         getEnv("SERVER_RUN_MODE", "debug"),
			HttpPort:        getEnvInt("SERVER_HTTP_PORT", 9001),
			HttpHost:        getEnv("SERVER_HTTP_HOST", "0.0.0.0"),
			ReadTimeout:     time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 60)) * time.Second,
			WriteTimeout:    time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 60)) * time.Second,
			MsgArchHttpPort: getEnvInt("SERVER_MSG_ARCH_PORT", 9002),
			MsgArchSrvHost:  getEnv("SERVER_MSG_ARCH_HOST", ""),
		},
		DB: DBConfig{
			Host:     getEnvRequired("DB_HOST"),
			User:     getEnvRequired("DB_USER"),
			Password: getEnvRequired("DB_PASSWORD"),
			Name:     getEnvRequired("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Redis: redisConfig{
			Host:        getEnvRequired("REDIS_HOST"),
			Password:    getEnv("REDIS_PASSWORD", ""),
			DBNumber:    getEnvInt("REDIS_DB_NUMBER", 0),
			IdleTimeout: time.Duration(getEnvInt("REDIS_IDLE_TIMEOUT", 300)) * time.Second,
			ReadTimeout: time.Duration(getEnvInt("REDIS_READ_TIMEOUT", 3)) * time.Second,
			DialTimeout: time.Duration(getEnvInt("REDIS_DIAL_TIMEOUT", 5)) * time.Second,
		},
		DelayQueue: delayQueueConfig{
			BucketSize:        getEnvInt("DELAY_QUEUE_BUCKET_SIZE", 3),
			BucketName:        getEnv("DELAY_QUEUE_BUCKET_NAME", "dq_bucket_%d"),
			QueueName:         getEnv("DELAY_QUEUE_NAME", "dq_queue_%s"),
			QueueBlockTimeout: getEnvInt("DELAY_QUEUE_BLOCK_TIMEOUT", 2),
		},
		Storage: StorageConfig{
			Type:            getEnv("STORAGE_TYPE", ""), // 留空禁用存储功能
			CdnURL:          getEnv("STORAGE_CDN_URL", ""),
			AccessKeyId:     getEnv("STORAGE_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("STORAGE_ACCESS_KEY_SECRET", ""),
			EndPoint:        getEnv("STORAGE_ENDPOINT", ""),
			Bucket:          getEnv("STORAGE_BUCKET", ""),
			SecretID:        getEnv("STORAGE_SECRET_ID", ""),
			SecretKey:       getEnv("STORAGE_SECRET_KEY", ""),
			BucketURL:       getEnv("STORAGE_BUCKET_URL", ""),
			LocalRootPath:   getEnv("STORAGE_LOCAL_ROOT_PATH", ""),
			ServerRootPath:  getEnv("STORAGE_SERVER_ROOT_PATH", ""),
		},
		WeWork: weWorkConfig{
			ExtCorpID:          getEnvRequired("WEWORK_EXT_CORP_ID"),
			ContactSecret:      getEnvRequired("WEWORK_CONTACT_SECRET"),
			CustomerSecret:     getEnvRequired("WEWORK_CUSTOMER_SECRET"),
			MainAgentID:        int64(getEnvInt("WEWORK_MAIN_AGENT_ID", 0)),
			MainAgentSecret:    getEnvRequired("WEWORK_MAIN_AGENT_SECRET"),
			CallbackToken:      getEnvRequired("WEWORK_CALLBACK_TOKEN"),
			CallbackAesKey:     getEnvRequired("WEWORK_CALLBACK_AES_KEY"),
			PriKeyPath:         getEnv("WEWORK_PRI_KEY_PATH", ""),
			MsgArchBatchSize:   getEnvInt("WEWORK_MSG_ARCH_BATCH_SIZE", 100),
			MsgArchTimeout:     getEnvInt("WEWORK_MSG_ARCH_TIMEOUT", 10),
			MsgArchProxy:       getEnv("WEWORK_MSG_ARCH_PROXY", ""),
			MsgArchProxyPasswd: getEnv("WEWORK_MSG_ARCH_PROXY_PASSWD", ""),
		},
		MingDaoYun: MingDaoYunConfig{
			APIBase:             getEnv("MINGDAOYUN_API_BASE", "https://api.mingdao.com"),
			AppKey:              getEnv("MINGDAOYUN_APP_KEY", ""),
			Sign:                getEnv("MINGDAOYUN_SIGN", ""),
			CustomerWorksheetID: getEnv("MINGDAOYUN_CUSTOMER_WORKSHEET_ID", ""),
		},
	}
	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired 获取必需的环境变量，如果不存在则 panic
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable not set: %s", key)
	}
	return value
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
