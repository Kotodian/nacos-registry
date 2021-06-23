package nacos_registry

import (
	"context"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"strconv"
)

type NacosRegistry struct {
	client *Client
}

type RegistryOptions struct {
	Addrs   []string
	Context context.Context
}

// Service 具体的服务
type Service struct {
	Namespace string
	// Name 服务名称
	Name string
	// Nodes 下面的所有可请求的节点
	Nodes []*Node
}

// Node 该服务下的节点
type Node struct {
	// 唯一值 主要是注册消息队列
	Id string
	// 地址 具体请求的地址
	Address string
}

func NewRegistry(client *Client) *NacosRegistry {
	return &NacosRegistry{
		client: client,
	}
}

func (n *NacosRegistry) Register(s *Service) error {
	param := vo.RegisterInstanceParam{}
	param.Ip, param.Port = n.splitIpPort(s)
	param.ServiceName = s.Name
	param.Enable = true
	param.Healthy = true
	param.Weight = 1.0
	param.Ephemeral = true
	param.GroupName = s.Namespace
	_, err := n.client.client.RegisterInstance(param)
	return err
}

func (n *NacosRegistry) DeRegister(s *Service) error {
	param := vo.DeregisterInstanceParam{}
	param.Ip, param.Port = n.splitIpPort(s)
	param.ServiceName = s.Name
	param.GroupName = s.Namespace
	_, err := n.client.client.DeregisterInstance(param)
	return err
}

func (n *NacosRegistry) ListNodes(s *Service) ([]*Node, error) {
	param := vo.SelectAllInstancesParam{
		ServiceName: s.Name,
		GroupName:   s.Namespace,
	}
	nodes, err := n.client.client.SelectAllInstances(param)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	s.Nodes = make([]*Node, 0)
	for _, node := range nodes {
		s.Nodes = append(s.Nodes, &Node{
			Id:      node.InstanceId,
			Address: n.joinIpPort(node.Ip, node.Port),
		})
	}
	return s.Nodes, nil
}

func (n *NacosRegistry) splitIpPort(s *Service) (string, uint64) {
	ip, portStr, _ := net.SplitHostPort(s.Nodes[0].Address)
	port, _ := strconv.ParseUint(portStr, 10, 64)
	return ip, port
}

func (n *NacosRegistry) joinIpPort(ip string, port uint64) string {
	portStr := strconv.FormatUint(port, 10)
	return net.JoinHostPort(ip, portStr)
}

func (n *NacosRegistry) Watch(namespace string) (Watcher, error) {
	return NewNacosWatcher(n, namespace)
}
