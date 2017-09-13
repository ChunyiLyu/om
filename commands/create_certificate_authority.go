package commands

import (
	"strconv"

	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/flags"
)

type CreateCertificateAuthority struct {
	service     certificateAuthoritiesService
	tableWriter tableWriter
	Options     struct {
		CertPem    string `long:"certificate-pem"  description:"certificate"`
		PrivateKey string `long:"private-key-pem"  description:"private key"`
	}
}

func NewCreateCertificateAuthority(service certificateAuthoritiesService, tableWriter tableWriter) CreateCertificateAuthority {
	return CreateCertificateAuthority{service: service, tableWriter: tableWriter}
}

func (c CreateCertificateAuthority) Execute(args []string) error {
	_, err := flags.Parse(&c.Options, args)
	if err != nil {
		return err
	}

	ca, err := c.service.Create(api.CertificateAuthorityBody{
		CertPem:       c.Options.CertPem,
		PrivateKeyPem: c.Options.PrivateKey,
	})
	if err != nil {
		return err
	}

	c.tableWriter.SetAutoWrapText(false)
	c.tableWriter.SetHeader([]string{"id", "issuer", "active", "created on", "expired on", "certicate pem"})
	c.tableWriter.Append([]string{ca.GUID, ca.Issuer, strconv.FormatBool(ca.Active), ca.CreatedOn, ca.ExpiresOn, ca.CertPEM})
	c.tableWriter.Render()
	return nil
}

func (c CreateCertificateAuthority) Usage() Usage {
	return Usage{
		Description:      "This authenticated command creates a certificate authority on the Ops Manager with the given cert and key",
		ShortDescription: "creates a certificate authority on the Opsman",
	}
}
