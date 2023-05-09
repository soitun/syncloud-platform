package access

import (
	"fmt"
	"github.com/syncloud/platform/config"
	"github.com/syncloud/platform/rest/model"
	"go.uber.org/zap"
	"net"
)

type UserConfig interface {
	IsRedirectEnabled() bool
	SetIpv4Enabled(enabled bool)
	SetIpv4Public(enabled bool)
	SetIpv6Enabled(enabled bool)
	SetPublicIp(publicIp *string)
	SetPublicPort(port *int)
	GetPublicIp() *string
	GetPublicPort() *int
	IsIpv6Enabled() bool
	IsIpv4Public() bool
	IsIpv4Enabled() bool
}

type Redirect interface {
	Update(ipv4 *string, port *int, ipv4Enabled bool, ipv4Public bool, ipv6Enabled bool) error
}

type Trigger interface {
	RunAccessChangeEvent() error
}

type NetworkInfo interface {
	IPv6() (*string, error)
	PublicIPv4() (*string, error)
}

type Response struct {
	Success bool    `json:"success"`
	Message *string `json:"message"`
}

type Probe interface {
	Probe(ip string, port int) error
}

type ExternalAddress struct {
	probe      Probe
	userConfig UserConfig
	redirect   Redirect
	trigger    Trigger
	network    NetworkInfo
	logger     *zap.Logger
}

func New(probe Probe, userConfig UserConfig, redirect Redirect, trigger Trigger, network NetworkInfo, logger *zap.Logger) *ExternalAddress {
	return &ExternalAddress{
		probe:      probe,
		userConfig: userConfig,
		redirect:   redirect,
		trigger:    trigger,
		network:    network,
		logger:     logger,
	}
}

func (a *ExternalAddress) Update(request model.Access) error {

	a.logger.Info(fmt.Sprintf("update ipv4 enabled: %v, ipv4 public: %v, ipv6 enabled: %v",
		request.Ipv4Enabled, request.Ipv4Public, request.Ipv6Enabled))

	ipv4 := request.Ipv4
	ipv4ToSave := ipv4
	if request.Ipv4Enabled {

		port := config.WebAccessPort
		if request.AccessPort != nil {
			port = *request.AccessPort
		}
		if ipv4 != nil {
			addr := net.ParseIP(*ipv4)
			if addr.To4() == nil {
				ipv4 = nil
				ipv4ToSave = nil
			}
		}

		if request.Ipv4Public {
			if ipv4 == nil {
				publicIp, err := a.network.PublicIPv4()
				if err != nil {
					return err
				}
				ipv4 = publicIp
			}
			err := a.probe.Probe(*ipv4, port)
			if err != nil {
				return err
			}
		}
	}

	if request.Ipv6Enabled {
		ipv6, err := a.network.IPv6()
		if err != nil {
			return err
		}
		err = a.probe.Probe(*ipv6, config.WebAccessPort)
		if err != nil {
			return err
		}
	}

	if a.userConfig.IsRedirectEnabled() {
		err := a.redirect.Update(
			ipv4,
			request.AccessPort,
			request.Ipv4Enabled,
			request.Ipv4Public,
			request.Ipv6Enabled)
		if err != nil {
			return err
		}
	}
	a.userConfig.SetIpv4Enabled(request.Ipv4Enabled)
	a.userConfig.SetIpv4Public(request.Ipv4Public)
	a.userConfig.SetPublicIp(ipv4ToSave)
	a.userConfig.SetIpv6Enabled(request.Ipv6Enabled)
	a.userConfig.SetPublicPort(request.AccessPort)

	return a.trigger.RunAccessChangeEvent()

}

func (a *ExternalAddress) Sync() error {

	if a.userConfig.IsRedirectEnabled() {
		err := a.redirect.Update(
			a.userConfig.GetPublicIp(),
			a.userConfig.GetPublicPort(),
			a.userConfig.IsIpv4Enabled(),
			a.userConfig.IsIpv4Public(),
			a.userConfig.IsIpv6Enabled())
		if err != nil {
			return err
		}
	}
	return nil
}
