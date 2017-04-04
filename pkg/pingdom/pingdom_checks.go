package pingdom

import "sync"

type pingdomChecks struct {
	dataMux *sync.Mutex
	data    map[string][]int
}

func newPingdomChecks() *pingdomChecks {
	return &pingdomChecks{
		dataMux: new(sync.Mutex),
		data:    make(map[string][]int),
	}
}

func (p *pingdomChecks) Get(checkName string) (ids []int) {
	p.dataMux.Lock()
	defer p.dataMux.Unlock()

	return p.data[checkName]
}

func (p *pingdomChecks) Add(checkName string, ids ...int) {
	p.dataMux.Lock()
	defer p.dataMux.Unlock()

	s := p.data[checkName]
	if s == nil {
		p.data[checkName] = ids
		return
	}
	p.data[checkName] = append(s, ids...)
}

func (p *pingdomChecks) Delete(checkName string, ids ...int) {
	p.dataMux.Lock()
	defer p.dataMux.Unlock()

	s := p.data[checkName]
	if s == nil {
		return
	}

	for _, toDelete := range ids {
		for i := len(s) - 1; i >= 0; i-- {
			if s[i] == toDelete {
				s = append(s[:i], s[i+1:]...)
			}
		}
	}

	p.data[checkName] = s
}
