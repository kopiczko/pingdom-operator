package pingdom

import (
	"fmt"

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

// Updates a HTTP check.
func (c *Operator) updateCheck(id int, checkSpec tpr.Spec) error {
	r, err := c.pclient.Checks.Read(id)
	if err != nil {
		return fmt.Errorf("reading check with id:%d: %v", id, err)
	}
	hc := pdom.HttpCheck{
		Name:                     r.Name,
		Hostname:                 r.Hostname,
		Resolution:               checkSpec.Resolution,
		SendNotificationWhenDown: r.SendNotificationWhenDown,
	}
	_, err = c.pclient.Checks.Update(id, &hc)
	return err
}

// Deletes the HTTP check.
func (c *Operator) deleteCheck(checkID int) error {
	_, err := c.pclient.Checks.Delete(checkID)
	return err
}
