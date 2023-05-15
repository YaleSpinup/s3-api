package route53

import (
	"github.com/YaleSpinup/s3-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	log "github.com/sirupsen/logrus"
)

// Route53 is a wrapper around the aws route53 service with some default config info
type Route53 struct {
	Service route53iface.Route53API
	Domains map[string]*common.Domain
}

// NewSession creates a new cloudfront session
func NewSession(sess *session.Session, account common.Account) Route53 {
	r := Route53{}
	if sess == nil {
		log.Infof("creating new aws session for route53 with key id %s in region %s", account.Akid, account.Region)
		sess = session.Must(session.NewSession(&aws.Config{
			Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
			Region:      aws.String(account.Region),
		}))
	}
	r.Service = route53.New(sess)
	r.Domains = account.Domains
	return r
}
