package inventory

import "testing"

func FuzzMCPConfigParsing(f *testing.F) {
	f.Add(`{"mcpServers":{"demo":{"command":"npx","args":["@example/server"],"env":{"API_TOKEN":"${API_TOKEN}"}}}}`)
	f.Add(`[mcp_servers.demo]
url = "http://127.0.0.1:8787/mcp"

[mcp_servers.demo.headers]
Authorization = "Bearer secret"
`)
	f.Add(`{"servers":{"broken":{"headers":{"X-API-Key":"inline-secret"},"args":[1,true,null]}}}`)

	f.Fuzz(func(t *testing.T, data string) {
		if len(data) > 4096 {
			t.Skip("cap fuzz input size for parser smoke coverage")
		}
		_, _ = readJSONServers([]byte(data))
		_, _ = readTOMLServers([]byte(data))
		_, _ = readYAMLServers([]byte(data))
	})
}
