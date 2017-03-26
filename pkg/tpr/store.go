package tpr

import "sync"

type storeKey struct {
	namespace, name string
}

type Store struct {
	dataMux *sync.Mutex
	data    map[storeKey]Spec
}

func NewStore() *Store {
	return &Store{
		dataMux: new(sync.Mutex),
		data:    make(map[storeKey]Spec),
	}
}

func (s *Store) Get(namespace, name string) (spec Spec, ok bool) {
	k := storeKey{namespace: namespace, name: name}
	s.dataMux.Lock()
	spec, ok = s.data[k]
	s.dataMux.Unlock()
	return
}

func (s *Store) set(check *PingdomCheck) {
	k := storeKey{namespace: check.Namespace, name: check.Name}
	s.dataMux.Lock()
	s.data[k] = check.Spec
	s.dataMux.Unlock()
}

func (s *Store) delete(check *PingdomCheck) {
	k := storeKey{namespace: check.Namespace, name: check.Name}
	s.dataMux.Lock()
	delete(s.data, k)
	s.dataMux.Unlock()
}
