package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

func NewDDAgent() *DDAgent {
	dd := &DDAgent{}
	for _, pattern := range []string{"/spans", "/v0.1/spans", "/v0.2/traces", "/v0.3/traces", "/v0.4/traces", "/v0.5/traces", "/v0.7/traces"} {
		dd.HandleFunc(pattern, handleWithPatternWrapper(pattern))
	}

	return dd
}

type DDAgent struct {
	http.ServeMux
}

func (dd *DDAgent) Start(addr string) {
	go func() {
		if err := http.ListenAndServe(addr, dd); err != nil {
			log.Fatalln(err.Error())
		}
	}()
}

func handleWithPatternWrapper(pattern string) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		tc := countTraces(req)
		if tc == 0 {
			resp.WriteHeader(http.StatusBadRequest)

			return
		}

		var err error
		switch pattern {
		case "/spans", "/v0.1/spans":
			var spans []pb.Span
			err = json.NewDecoder(req.Body).Decode(&spans)
		case "/v0.2/traces", "/v0.3/traces", "/v0.4/traces":

		case "/v0.5/traces":
		case "/v0.7/traces":
		}
		if err != nil {
			log.Println(err.Error())
			resp.WriteHeader(http.StatusBadRequest)
		} else {
			// todo: reply ok
		}
	}
}

func decodeRequest(req *http.Request) (pb.Traces, error) {
	var traces = pb.Traces{}
	switch mt := getMetaType(req, "application/json"); mt {
	case "application/msgpack":
		buf := getBuffer()
		defer putBuffer(buf)

		if _, err := io.Copy(buf, req.Body); err != nil {
			return nil, err
		}
		if _, err := traces.UnmarshalMsg(buf.Bytes()); err != nil {
			return nil, err
		}
	case "application/json", "test/json", "":
		if err := json.NewDecoder(req.Body).Decode(&traces); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognized media type: %s", mt)
	}

	return traces, nil
}

func countTraces(req *http.Request) int {
	v := req.Header.Get("X-Datadog-Trace-Count")
	if v == "" || v == "0" {
		return 0
	}
	if c, err := strconv.Atoi(v); err != nil {
		return 0
	} else {
		return c
	}
}
