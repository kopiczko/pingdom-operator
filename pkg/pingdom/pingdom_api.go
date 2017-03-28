package pingdom

import (
	"github.com/rossf7/pingdom-operator/pkg/tpr"
	pdom "github.com/russellcardullo/go-pingdom/pingdom"
)

var (
	defaultCheckSpec = tpr.Spec{
		Resolution: 1,
	}
)

// Creates a HTTP check for the host and returns the Pingdom ID.
func (c *Operator) createCheck(host string, checkSpec tpr.Spec) (int, error) {
	hc := pdom.HttpCheck{Name: host, Hostname: host, Resolution: checkSpec.Resolution}
	check, err := c.pclient.Checks.Create(&hc)
	if err != nil {
		return -1, err
	}
	return check.ID, nil
}

// Deletes the HTTP check.
func (c *Operator) deleteCheck(checkID int) error {
	_, err := c.pclient.Checks.Delete(checkID)
	return err
}
