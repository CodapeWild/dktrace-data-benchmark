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

var (
	V01            = "v01"
	V02            = "v02"
	V03            = "v03"
	V04            = "v04"
	V05            = "v05"
	V07            = "v07"
	PatternVersion = map[string]string{
		"/spans": V01, "/v0.1/spans": V01,
		"/v0.2/traces": V02,
		"/v0.3/traces": V03,
		"/v0.4/traces": V04,
		"/v0.5/traces": V05,
		"/v0.7/traces": V07,
	}
)

func NewDDAgent(ampf *DDAmplifier) *DDAgent {
	if ampf == nil {
		log.Fatalln("traces amplifier for ddtrace agent can not be nil")
	}

	dd := &DDAgent{}
	for pattern, version := range PatternVersion {
		dd.HandleFunc(pattern, handleTracesWrapper(version, ampf))
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

func handleTracesWrapper(version string, ampf *DDAmplifier) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		tc := countTraces(req)
		if tc == 0 {
			resp.WriteHeader(http.StatusOK)

			return
		}

		var (
			traces pb.Traces
			err    error
		)
		switch version {
		case V01:
			var spans []*pb.Span
			if err = json.NewDecoder(req.Body).Decode(&spans); err == nil {
				traces = append(traces, pb.Trace(spans))
			}
		case V02, V03, V04:
			traces, err = decodeRequest(req)
		case V05:
			buf := getBuffer()
			defer putBuffer(buf)

			if _, err = io.Copy(buf, req.Body); err == nil {
				err = traces.UnmarshalMsgDictionary(buf.Bytes())
			}
		case V07:
			buf := getBuffer()
			defer putBuffer(buf)

			if _, err = io.Copy(buf, req.Body); err == nil {
				_, err = traces.UnmarshalMsg(buf.Bytes())
			}
		}

		// reply ok or error based on parameter err
		reply(version, resp, err)
		if err != nil {
			log.Println(err.Error())

			return
		} else if len(traces) == 0 {
			log.Println("empty traces")

			return
		}

		log.Printf("%v", traces)
		ampf.SendTraces(traces)
	}
}

func reply(version string, resp http.ResponseWriter, err error) {
	if err == nil {
		resp.WriteHeader(http.StatusOK)
		switch version {
		case V01, V02, V03:
			io.WriteString(resp, "OK\n")
		default:
			resp.Header().Set("Content-Type", "application/json")
			resp.Write([]byte("{}"))
		}
	} else {
		resp.WriteHeader(http.StatusBadRequest)
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

type DDAmplifier struct {
	addr               string
	threads, repeat    int
	expectedSpansCount int
	receivedSpansCount int
	traces             pb.Traces
	tc                 chan pb.Traces
	closer             chan struct{}
}

func (ddampf *DDAmplifier) StartThreads(ctx context.Context) {
	if err := ctx.Err(); err != nil {
		log.Println(err.Error())

		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					log.Printf("error: %s", err.Error())
				} else {
					log.Println("ddtrace amplifier context done")
				}
			case <-ddampf.closer:
				log.Println("ddtrace amplifier closed")
			case traces := <-ddampf.tc:
				ddampf.runThreads(traces)
				log.Println("ddtrace amplifier finished sending traces")
			}
		}
	}()
}

func (ddampf *DDAmplifier) SendTraces(traces pb.Traces) {
	ddampf.traces = append(ddampf.traces, traces...)
	for i := range ddampf.traces {
		ddampf.receivedSpansCount += len(ddampf.traces[i])
	}
	if ddampf.receivedSpansCount >= ddampf.expectedSpansCount {
		ddampf.tc <- ddampf.traces
	}
}

func (ddampf *DDAmplifier) Close() {
	select {
	case <-ddampf.closer:
		return
	default:
		close(ddampf.closer)
	}
}

func (ddampf *DDAmplifier) runThreads(traces pb.Traces) {
	clnt := http.Client{Transport: newSingleHostTransport()}
	for i := 1; i <= ddampf.threads; i++ {
		dupli := duplicateDDTraces(traces)
		go func(traces pb.Traces, i int) {
			for j := 1; j <= ddampf.repeat; i++ {
				if bts, err := traces.MarshalMsg(nil); err != nil {
					log.Println(err.Error())
				} else {
					req, err := http.NewRequest(http.MethodPost, ddampf.addr, bytes.NewBuffer(bts))
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

func NewDDAmplifier(ip string, port int, path string, expectedSpansCount, threads, repeat int) *DDAmplifier {
	return &DDAmplifier{
		addr:               fmt.Sprintf("http://%s:%s%s", ip, port, path),
		expectedSpansCount: expectedSpansCount,
		threads:            threads,
		repeat:             repeat,
		tc:                 make(chan pb.Traces),
		closer:             make(chan struct{}),
	}
}

func BuildDDAgentForWork(agentAddress string, endpointIP string, endpointPort int, endpointPath string, expectedSpansCount, threads, repeat int) context.CancelFunc {
	ctx, canceler := context.WithCancel(context.TODO())

	ampf := NewDDAmplifier(endpointIP, endpointPort, endpointPath, expectedSpansCount, threads, repeat)
	ampf.StartThreads(ctx)

	agent := NewDDAgent(ampf)
	agent.Start(agentAddress)

	return canceler
}
