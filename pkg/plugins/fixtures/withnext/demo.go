package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

// config is the plugin configuration.
//
//   - header: the value of the X-Plugin header set on the request given to next.
//   - upperResponse: when true, the response body of next is read and upper-cased.
//   - panicOn: a request path for which handle_request panics (to test instance replacement).
//   - echoBody: when true, the request body is read and sent back as the response (no next).
//   - loopOn: a request path for which handle_request never returns (to test the call timeout).
type config struct {
	Header        string `json:"header"`
	UpperResponse bool   `json:"upperResponse"`
	PanicOn       string `json:"panicOn"`
	EchoBody      bool   `json:"echoBody"`
	LoopOn        string `json:"loopOn"`
}

var cfg config

// Built as a reactor (main never runs, the handlers are registered in init) with
// tinygo build -o plugin.wasm -buildmode=c-shared -scheduler=none --no-debug -target=wasip1 ./demo.go
func main() {}

func init() {
	err := json.Unmarshal(handler.Host.GetConfig(), &cfg)
	if err != nil {
		handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not load config %v", err))
		os.Exit(1)
	}

	// Setting response headers after next requires the response to be buffered.
	handler.Host.EnableFeatures(api.FeatureBufferResponse)

	handler.HandleRequestFn = handleRequest
	handler.HandleResponseFn = handleResponse
}

// counter numbers the requests seen by this instance: handle_response gets it back as reqCtx.
var counter uint32

func handleRequest(req api.Request, resp api.Response) (bool, uint32) {
	if cfg.PanicOn != "" && req.GetURI() == cfg.PanicOn {
		panic("requested panic")
	}

	if cfg.LoopOn != "" && req.GetURI() == cfg.LoopOn {
		for {
			counter++
		}
	}

	if cfg.EchoBody {
		var body []byte
		buf := make([]byte, 4096)
		for {
			n, eof := req.Body().Read(buf)
			body = append(body, buf[:n]...)
			if eof {
				break
			}
		}
		resp.Headers().Set("X-Body-Length", strconv.Itoa(len(body)))
		resp.Body().Write(body)

		return false, 0
	}

	counter++
	req.Headers().Set("X-Plugin", cfg.Header)
	req.Headers().Set("X-Plugin-Seq", strconv.Itoa(int(counter)))

	return true, counter
}

func handleResponse(reqCtx uint32, _ api.Request, resp api.Response, isError bool) {
	resp.Headers().Set("X-Plugin-Resp", strconv.Itoa(int(reqCtx)))
	if isError {
		resp.Headers().Set("X-Plugin-Next-Error", "true")
	}

	if !cfg.UpperResponse {
		return
	}

	var body []byte
	buf := make([]byte, 4096)
	for {
		n, eof := resp.Body().Read(buf)
		body = append(body, buf[:n]...)
		if eof {
			break
		}
	}
	resp.Body().WriteString(strings.ToUpper(string(body)))
}
