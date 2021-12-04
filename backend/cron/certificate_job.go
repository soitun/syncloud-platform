package cron

import (
	"github.com/syncloud/platform/config"
	"github.com/syncloud/platform/network"
)

type CertificateJob struct {
	userConfig   *config.UserConfig
	networkIface *network.Interface
}

func NewCertificateJob(userConfig *config.UserConfig, networkIface *network.Interface) *CertificateJob {
	return &CertificateJob{
		userConfig:   userConfig,
		networkIface: networkIface,
	}
}

func (j *CertificateJob) Run() error {

	localIpv4, err := j.networkIface.LocalIPv4()
	if err != nil {
		return err
	}

	ipv6Available := true
	_, err = j.networkIface.IPv6()
	if err != nil {
		ipv6Available = false
	}

	generateRealCertificate := true
	if j.userConfig.IsRedirectEnabled() {
		if !j.userConfig.GetExternalAccess() {
			if localIpv4.IsPrivate() && !ipv6Available {
				generateRealCertificate = false
			}
		}
	}
	if generateRealCertificate {
		//injector.tls.generate_real_certificate()
	}

	return err
}
