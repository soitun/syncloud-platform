package rest

import (
	"github.com/syncloud/platform/cert"
	"net/http"
)

type Certificate struct {
	infoReader CertificateInfoReader
	logReader  *cert.Reader
}

type CertificateInfoReader interface {
	ReadCertificateInfo() *cert.Info
}

func NewCertificate(infoReader CertificateInfoReader, certLogReader *cert.Reader) *Certificate {
	return &Certificate{
		infoReader: infoReader,
		logReader:  certLogReader,
	}
}

func (c *Certificate) Certificate(_ *http.Request) (interface{}, error) {
	return c.infoReader.ReadCertificateInfo(), nil
}

func (c *Certificate) CertificateLog(_ *http.Request) (interface{}, error) {
	return c.logReader.Read(), nil
}