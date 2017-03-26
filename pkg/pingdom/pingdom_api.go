package pingdom

import (
	"github.com/rossf7/pingdom-operator/pkg/tpr"
	pdom "github.com/russellcardullo/go-pingdom/pingdom"
)

var (
	defaultCheckSpec = tpr.Spec{
		RetryInterval: 1,
	}
)

// Creates a HTTP check for the host and returns the Pingdom ID.
func (c *Operator) createCheck(host string, checkSpec tpr.Spec) (int, error) {
	hc := pdom.HttpCheck{Name: host, Hostname: host, Resolution: checkSpec.RetryInterval}
	check, err := c.pclient.Checks.Create(&hc)
	if err != nil {
		log.Errorf("Failed to create check for host %s: %v", host, err)
		return -1, err
	}

	return check.ID, err
}

// Deletes the HTTP check.
func (c *Operator) deleteCheck(checkID int) error {
	_, err := c.pclient.Checks.Delete(checkID)
	if err != nil {
		log.Errorf("Failed to delete check %d: %v", checkID, err)
	}

	return err
}
