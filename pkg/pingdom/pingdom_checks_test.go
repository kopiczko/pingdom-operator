package pingdom

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPingdomChecks(t *testing.T) {
	p := newPingdomChecks()
	assert.Equal(t, []int(nil), p.Get("a"))
	p.Add("a", 1, 3, 5, 7)
	assert.Equal(t, []int{1, 3, 5, 7}, p.Get("a"))
	p.Delete("a", 3, 7)
	assert.Equal(t, []int{1, 5}, p.Get("a"))
	p.Add("a", 7, 8)
	assert.Equal(t, []int{1, 5, 7, 8}, p.Get("a"))
	p.Add("a", 9)
	assert.Equal(t, []int{1, 5, 7, 8, 9}, p.Get("a"))
	p.Delete("a", 7)
	assert.Equal(t, []int{1, 5, 8, 9}, p.Get("a"))
}
