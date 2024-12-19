package duck

import (
	"fmt"
	"github.com/YaleSpinup/s3-api/iam"

	"github.com/micromdm/plist"
	log "github.com/sirupsen/logrus"
)

type DotDuck struct {
	Protocol string `plist:"Protocol"`
	Provider string `plist:"Provider"`
	Nickname string `plist:"Nickname"`
	Hostname string `plist:"Hostname"`
	Port     string `plist:"Port"`
	Path     string `plist:"Path"`
	WebURL   string `plist:"Web URL"`
}

func DefaultDuck(name string, path string) *DotDuck {
	fmtPath := fmt.Sprintf("/%s", name)
	fmtNickname := fmt.Sprintf("Spinup - %s", name)

	if path != "/" {
		path = iam.RemoveCappingSlashes(path)
		fmtPath = fmt.Sprintf("/%s/%s", name, path)
		fmtNickname = fmt.Sprintf("Spinup - %s/%s", name, path)
	}

	return &DotDuck{
		Protocol: "s3",
		Provider: "iterate GmbH",
		Nickname: fmtNickname,
		Hostname: "s3.amazonaws.com",
		Port:     "443",
		Path:     fmtPath,
		WebURL:   fmt.Sprintf("s3://%s/", name),
	}
}

// Generate is a simple bookmark file generator for cyberduck
func (d *DotDuck) Generate() ([]byte, error) {
	log.Debug("generating cyberduck bookmark file")

	doc, err := plist.MarshalIndent(d, "  ")
	if err != nil {
		return nil, err
	}

	return doc, nil
}
