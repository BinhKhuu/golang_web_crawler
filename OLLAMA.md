# Run locally  ( not docker )

1. terminal 1: ollama serve
2. terminal 2: ollama run mistral:latest 

# list llms installed

1. ollama list

# Ollama model comparision
1. mistral:latest
	- this is the fastest model so far for data extraction
2. qwen3.5:latest
	- this is the slowest model so far for data extraction
	- resource intensive 
	- results not as accurate as mistral

# Setting model in in application
internal/llm/llm.go

``` go
const (
	Model        = "mistral:latest"
	MaxMemoryMBs = 16384
)
```

# ollama golang package new port 
The official Ollama Go package provides a NewClient function that allows you to explicitly specify the host and port without relying on environment variables. 
Using api.NewClient
Instead of ClientFromEnvironment(), you can use NewClient, which requires a parsed *url.URL and an *http.Client. 

package main
import (
	"net/http"
	"net/url"
	"github.com/ollama/ollama/api"
)
func main() {
	// 1. Define your Docker port (e.g., 11435)
	rawURL := "http://localhost:11435"
	u, _ := url.Parse(rawURL)

	// 2. Create the client explicitly
	client := api.NewClient(u, http.DefaultClient)

	// Now 'client' points directly to your Dockerized instance
}

Comparison of Client Constructors

* ClientFromEnvironment(): Automatically reads the [OLLAMA_HOST environment variable](https://github.com/ollama/ollama/blob/main/api/client.go). If unset, it defaults to 127.0.0.1:11434.
* NewClient(base *url.URL, http *http.Client): Gives you full programmatic control over the connection string (scheme, host, and port) and allows you to inject custom HTTP settings (like timeouts or proxies).

