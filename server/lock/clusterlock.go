package lock

import "sync"

type Map struct {
	m sync.Map
}

func New() *Map {
	return &Map{}
}

func (m *Map) getOrCreate(cluster string) *sync.RWMutex {
	v, _ := m.m.LoadOrStore(cluster, &sync.RWMutex{})
	return v.(*sync.RWMutex)
}

func (m *Map) RLock(cluster string) {
	if cluster == "" {
		return
	}
	m.getOrCreate(cluster).RLock()
}

func (m *Map) RUnlock(cluster string) {
	if cluster == "" {
		return
	}
	m.getOrCreate(cluster).RUnlock()
}

func (m *Map) TryLock(cluster string) bool {
	if cluster == "" {
		return true
	}
	return m.getOrCreate(cluster).TryLock()
}

func (m *Map) Unlock(cluster string) {
	if cluster == "" {
		return
	}
	m.getOrCreate(cluster).Unlock()
}
