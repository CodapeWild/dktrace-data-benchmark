package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/trace/pb"
)

const (
	v01 = "v01"
	v02 = "v02"
	v03 = "v03"
	v04 = "v04"
	v05 = "v05"
	v07 = "v07"
)

func NewDDAgent() *DDAgent {
	dd := &DDAgent{}
	for _, pattern := range []string{"/spans", "/v0.1/spans", "/v0.2/traces", "/v0.3/traces", "/v0.4/traces", "/v0.5/traces", "/v0.7/traces"} {
		switch pattern {
		case "/spans", "/v0.1/spans":
			handleTracesWrapper(v01)
		case "/v0.2/traces":
			handleTracesWrapper(v02)
		case "/v0.3/traces":
			handleTracesWrapper(v03)
		case "/v0.4/traces":
			handleTracesWrapper(v04)
		case "/v0.5/traces":
			handleTracesWrapper(v05)
		case "/v0.7/traces":
			handleTracesWrapper(v07)
		default:
			log.Fatalln("unrecognized URL pattern for DDTrace")
		}
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

func handleTracesWrapper(version string) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		tc := countTraces(req)
		if tc == 0 {
			resp.WriteHeader(http.StatusBadRequest)

			return
		}

		var (
			traces pb.Traces
			err    error
		)
		switch version {
		case v01:
			var spans []*pb.Span
			if err = json.NewDecoder(req.Body).Decode(&spans); err == nil {
				traces = append(traces, pb.Trace(spans))
			}
		case v02, v03, v04:
			traces, err = decodeRequest(req)
		case v05:
			buf := getBuffer()
			defer putBuffer(buf)

			if _, err = io.Copy(buf, req.Body); err == nil {
				err = traces.UnmarshalMsgDictionary(buf.Bytes())
			}
		case v07:
			buf := getBuffer()
			defer putBuffer(buf)

			if _, err = io.Copy(buf, req.Body); err == nil {
				_, err = traces.UnmarshalMsg(buf.Bytes())
			}
		}
		reply(version, resp, err)

		if err == nil && len(traces) == 0 {

		}
	}
}

func decodeRequest(req *http.Request) (pb.Traces, error) {
	var traces pb.Traces
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

func reply(version string, resp http.ResponseWriter, err error) {
	if err == nil {
		resp.WriteHeader(http.StatusOK)
		switch version {
		case v01, v02, v03:
			io.WriteString(resp, "OK\n")
		default:
			resp.Header().Set("Content-Type", "application/json")
			resp.Write([]byte("{}"))
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
		log.Println(err.Error())
	}
}

func NewDDAmplifier(version string, ip string, port int, path string, threads, repeat int, spanCount int) *DDAmplifier {
	return &DDAmplifier{
		version:   version,
		addr:      fmt.Sprintf("http://%s:%s%s"),
		threads:   threads,
		repeat:    repeat,
		spanCount: spanCount,
		tc:        make(chan pb.Traces),
		closer:    make(chan struct{}),
	}
}

type DDAmplifier struct {
	version         string
	addr            string
	threads, repeat int
	spanCount       int
	traces          pb.Traces
	tc              chan pb.Traces
	closer          chan struct{}
}

func (ddamp *DDAmplifier) putTraces(traces pb.Traces) {
	ddamp.traces = append(ddamp.traces, traces...)
	var c int
	for i := range ddamp.traces {
		c += len(ddamp.traces[i])
	}
	if c >= ddamp.spanCount {
		ddamp.tc <- ddamp.traces
	}
}

func (ddamp *DDAmplifier) StartThreads(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		log.Println(err.Error())

		return
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			log.Printf("error: %s", err.Error())
		} else {
			log.Println("ddtrace amplifier context done")
		}
	case <-ddamp.closer:
		log.Println("ddtrace amplifier closed")
	case traces := <-ddamp.tc:
		ddamp.runThreads(traces)
		log.Println("ddtrace amplifier finished sending traces")
	}
}

func (ddamp *DDAmplifier) Close() {
	select {
	case <-ddamp.closer:
		return
	default:
		close(ddamp.closer)
	}
}

func (ddamp *DDAmplifier) runThreads(traces pb.Traces) {
	clnt := http.Client{Transport: newSingleHostTransport()}
	for i := 1; i <= ddamp.threads; i++ {
		dupli := duplicateDDTraces(traces)
		go func(traces pb.Traces, i int) {
			for j := 1; j <= ddamp.repeat; i++ {
				if bts, err := traces.MarshalMsg(nil); err != nil {
					log.Println(err.Error())
				} else {
					req, err := http.NewRequest(http.MethodPost, ddamp.addr, bytes.NewBuffer(bts))
					if err != nil {
						log.Fatalln(err)
					}
					resp, err := clnt.Do(req)
					if err != nil {
						log.Println(err.Error())
					} else {
						log.Printf("thread %d send %d times status: %s", i, j, resp.Status)
						resp.Body.Close()
					}
				}
				changeIDs(traces)
			}
		}(dupli, i)
	}
}

func duplicateDDTraces(traces pb.Traces) pb.Traces {
	buf := getBuffer()
	defer putBuffer(buf)

	if err := json.NewEncoder(buf).Encode(traces); err != nil {
		log.Fatalln(err.Error())
	}

	var dupli *pb.Traces = &pb.Traces{}
	if err := json.NewDecoder(buf).Decode(dupli); err != nil {
		log.Fatalln(err.Error())
	}

	return *dupli
}

func changeIDs(traces pb.Traces) {
	for i := range traces {
		var newtid = rand.Uint64()
		for j := range traces[i] {
			traces[i][j].TraceID = newtid
			var (
				newcid = rand.Uint64()
				oldcid = traces[i][j].SpanID
			)
			traces[i][j].SpanID = newcid
			for k := j + 1; k < len(traces[i]); k++ {
				if traces[i][k].ParentID == oldcid {
					traces[i][k].ParentID = newcid
					break
				}
			}
		}
	}
}
