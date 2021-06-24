package nacos_registry

import (
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"net/url"
	"strconv"
)

type Client struct {
	client    naming_client.INamingClient
	namespace string
}

type Config struct {
	// nacos client config
	clientConfig constant.ClientConfig
	// naocs server configs
	serverConfigs []constant.ServerConfig
}

func NewClientConfig(namespace string, timeout uint64, urls ...url.URL) *Config {
	if len(urls) == 0 {
		return nil
	}
	config := new(Config)
	// 初始化 nacos client配置
	clientConfig := constant.NewClientConfig(
		constant.WithTimeoutMs(timeout),
		constant.WithNamespaceId(namespace),
		constant.WithUsername("nacos"),
		constant.WithPassword("nacos"),
		constant.WithBeatInterval(1000),
		constant.WithNotLoadCacheAtStart(true),
	)
	config.clientConfig = *clientConfig
	// 初始化nacos server配置
	config.serverConfigs = make([]constant.ServerConfig, 0)
	for _, u := range urls {
		port, _ := strconv.ParseUint(u.Port(), 10, 64)
		config.serverConfigs = append(config.serverConfigs, *constant.NewServerConfig(u.Hostname(), port))
	}
	return config
}

func NewClientFromConfig(config *Config) (*Client, error) {
	client := new(Client)
	namingClient, err := clients.CreateNamingClient(map[string]interface{}{
		constant.KEY_SERVER_CONFIGS: config.serverConfigs,
		constant.KEY_CLIENT_CONFIG:  config.clientConfig,
	})
	if err != nil {
		return nil, err
	}
	client.client = namingClient
	client.namespace = config.clientConfig.NamespaceId
	return client, err
}
