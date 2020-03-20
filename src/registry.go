package main

const (
	serviceAbiDecoder     = "abiDecoder"
	serviceConfig         = "config"
	serviceDatabase       = "db"
	serviceFetchEvent     = "fetchEvent"
	serviceScraper        = "scraper"
	serviceSessionManager = "sessionManager"
)

type Registry struct {
	objects map[string]interface{}
}

func newRegistry() *Registry {
	return &Registry{make(map[string]interface{})}
}

func (r *Registry) set(key string, value interface{}) {
	r.objects[key] = value
}

func (r *Registry) get(key string) interface{} {
	return r.objects[key]
}

func (r *Registry) clean() {
	for k := range r.objects {
		delete(r.objects, k)
	}
}
