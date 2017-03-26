package tpr

import "sync"

type Store struct {
	dataMux *sync.Mutex
	data    map[string]Spec
}

func NewStore() *Store {
	return &Store{
		dataMux: new(sync.Mutex),
		data:    make(map[string]Spec),
	}
}

func (s *Store) Get(name string) (spec Spec, ok bool) {
	s.dataMux.Lock()
	spec, ok = s.data[name]
	s.dataMux.Unlock()
	return
}

func (s *Store) set(check *PingdomCheck) {
	s.dataMux.Lock()
	s.data[check.Name] = check.Spec
	s.dataMux.Unlock()
}

func (s *Store) delete(check *PingdomCheck) {
	s.dataMux.Lock()
	delete(s.data, check.Name)
	s.dataMux.Unlock()
}
