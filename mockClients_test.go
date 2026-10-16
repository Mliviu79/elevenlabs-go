package elevenlabs

import "time"

func NewMockClient(baseURL, apiKey string, reqTimeout time.Duration, opts ...ClientOption) *Client {
	c := NewClient(apiKey, reqTimeout, opts...)
	c.baseURL = baseURL
	return c
}

func MockDefaultClient(baseURL string) *Client {
	getDefaultClient()
	defaultClient.baseURL = baseURL
	return defaultClient
}
