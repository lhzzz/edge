package client

type EdgeClient interface {
	Connect(address string) error
}

func NewEdgeClient() EdgeClient {
	return &websocketClient{}
}
