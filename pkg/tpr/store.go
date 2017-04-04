package tpr

import "sync"

type StoreEventHandler interface {
	OnSet(namespace, name string, spec Spec)
	OnDelete(namespace, name string, spec Spec)
}

type StoreEventHandlerFuncs struct {
	SetFunc, DeleteFunc func(namespace, name string, spec Spec)
}

func (s StoreEventHandlerFuncs) OnSet(namespace, name string, spec Spec) {
	if s.SetFunc != nil {
		s.SetFunc(namespace, name, spec)
	}
}

func (s StoreEventHandlerFuncs) OnDelete(namespace, name string, spec Spec) {
	if s.DeleteFunc != nil {
		s.DeleteFunc(namespace, name, spec)
	}
}

type storeKey struct {
	namespace, name string
}

type Store struct {
	dataMux *sync.Mutex
	data    map[storeKey]Spec

	Handler StoreEventHandler
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
	if s.Handler != nil {
		s.Handler.OnSet(k.namespace, k.name, check.Spec)
	}
}

func (s *Store) delete(check *PingdomCheck) {
	k := storeKey{namespace: check.Namespace, name: check.Name}
	s.dataMux.Lock()
	delete(s.data, k)
	s.dataMux.Unlock()
	if s.Handler != nil {
		s.Handler.OnDelete(k.namespace, k.name, check.Spec)
	}
}
