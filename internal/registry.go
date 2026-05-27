package internal

import "sync"

type OryPolisClient struct {
	HTTP    *polisHTTPClient
	BaseURL string
}

var (
	clientMu       sync.RWMutex
	clientRegistry = map[string]*OryPolisClient{}
)

func RegisterClient(name string, client *OryPolisClient) {
	clientMu.Lock()
	defer clientMu.Unlock()
	clientRegistry[name] = client
}

func GetClient(name string) (*OryPolisClient, bool) {
	clientMu.RLock()
	defer clientMu.RUnlock()
	client, ok := clientRegistry[name]
	return client, ok
}

func UnregisterClient(name string) {
	clientMu.Lock()
	defer clientMu.Unlock()
	delete(clientRegistry, name)
}
