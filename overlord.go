package overlord

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"github.com/cocaine/cocaine-framework-go/vendor/src/github.com/ugorji/go/codec"
)

const (
	utilitySession = 1
	handshake      = 0

	invoke = 0
	chunk  = 0
	_error = 1
	close  = 2
)

type sessions map[uint64]chan *cocaine.Message

type logger struct{}

func (l *logger) Write(p []byte) (int, error) {
	fmt.Printf("%s", p)
	return len(p), nil
}

// Overlord simple mock proxy. It starts a worker and HTTP server
// to send requests to the worker.
type Overlord struct {
	cfg *Config

	mu       sync.Mutex
	counter  uint64
	sessions sessions

	conn net.Conn
}

// Config for Overlord
type Config struct {
	Slave          string
	Locator        string
	HTTPEndpoint   string
	StartUpTimeout time.Duration
}

// NewOverlord creates new Overlord with given config
func NewOverlord(cfg *Config) (*Overlord, error) {
	olord := &Overlord{
		cfg: cfg,

		counter:  10,
		sessions: make(sessions),
	}
	return olord, nil
}

// Start launches the worker, HTTP server. It's the only method
// user has to call.
func (o *Overlord) Start() error {
	appname := "testapp"
	socketPath := path.Join(
		os.TempDir(),
		fmt.Sprintf("%s.%d.sock", appname, os.Getpid()),
	)

	args := []string{
		path.Base(o.cfg.Slave),
		"--app", appname,
		"--locator", o.cfg.Locator,
		"--uuid", uuid.New(),
		"--endpoint", socketPath,
		"--protocol", "1",
	}

	cmd := exec.Cmd{
		Path: o.cfg.Slave,

		Args: args,
		// inside container worker has that cwd
		Dir: "/",

		Stdout: &logger{},
		Stderr: &logger{},
	}

	log.Printf("listening %s", socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer ln.Close()

	go func() {
		http.HandleFunc("/", o.handleHTTPRequest)

		log.Printf("starting HTTP server: %s", o.cfg.HTTPEndpoint)
		// ToDo: make it closable
		if err := http.ListenAndServe(o.cfg.HTTPEndpoint, nil); err != nil {
			log.Panicf("%v", err)
		}
	}()

	log.Printf("starting worker %s, work dir: %s", cmd.Path, cmd.Dir)
	log.Printf("locator %s", o.cfg.Locator)
	log.Printf("args: %v", args)
	if len(cmd.Env) > 0 {
		log.Println("with ENV:")
		for _, envvar := range cmd.Env {
			log.Println(envvar)
		}
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := o.acceptConnect(ln); err != nil {
		return err
	}

	return cmd.Wait()
}

func (o *Overlord) acceptConnect(ln net.Listener) error {
	if unixLn, ok := ln.(*net.UnixListener); ok {
		log.Printf("set acception deadline: %v", o.cfg.StartUpTimeout)
		unixLn.SetDeadline(time.Now().Add(o.cfg.StartUpTimeout))
	}

	conn, err := ln.Accept()
	if err != nil {
		log.Printf("accept errror %v", err)
		return err
	}
	o.conn = conn
	go o.handleConnection(o.conn)
	return nil
}

func (o *Overlord) handleConnection(conn io.ReadWriteCloser) {
	defer conn.Close()
	decoder := codec.NewDecoder(conn, hAsocket)
	encoder := codec.NewEncoder(conn, hAsocket)
	for {
		var msg cocaine.Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("protocol decoder error %v", err)
			return
		}
		switch msg.Session {
		case utilitySession:
			if msg.MsgType == handshake {
				log.Println("replying to heartbeat")
				encoder.Encode(msg)
			}
		default:
			o.mu.Lock()
			if ch, ok := o.sessions[msg.Session]; ok {
				select {
				case ch <- &msg:
				case <-time.After(time.Second * 2):
					log.Println("didn't send reply during 2 seconds")
				}
			} else {
				log.Printf("Invalid session %v", msg)
			}
			o.mu.Unlock()
		}
	}
}

func (o *Overlord) handleHTTPRequest(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Add("X-Powered-By", "Cocaine")

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		w.Header().Add("X-Error-Generated-By", "Cocaine")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "unable to read the whole body")
		return
	}

	// method uri 1.1 headers body
	headers := make([][2]string, 0, len(req.Header))
	for header, values := range req.Header {
		for _, val := range values {
			headers = append(headers, [2]string{header, val})
		}
	}

	var task []byte
	codec.NewEncoderBytes(&task, hAsocket).Encode([]interface{}{
		req.Method,
		req.URL.RequestURI(),
		fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor),
		headers,
		body,
	})

	channel := make(chan *cocaine.Message, 5)
	o.mu.Lock()
	o.counter++
	counter := o.counter
	o.sessions[o.counter] = channel
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		delete(o.sessions, counter)
	}()

	enqueueTask(counter, task, o.conn)
	var first = true
FOR:
	for msg := range channel {
		switch msg.MsgType {
		case chunk:
			payload, ok := msg.Payload[0].([]byte)
			if !ok {
				log.Panicf("invalid response data, must be []byte")
				continue FOR
			}

			if first {
				first = false
				var res struct {
					Code    int
					Headers [][2]string
				}
				codec.NewDecoderBytes(payload, hAsocket).Decode(&res)
				for _, header := range res.Headers {
					w.Header().Add(header[0], header[1])
				}
				w.WriteHeader(res.Code)
				continue FOR
			}
			w.Write(payload)
		case _error:
			log.Println("error message from worker")
			w.Header().Add("X-Error-Generated-By", "Cocaine")
			w.WriteHeader(http.StatusInternalServerError)
			var msgerr struct {
				CodeInfo [2]int
				Message  string
			}
			if err := convertPayload(msg.Payload, &msgerr); err != nil {
				fmt.Fprintf(w, "unable to decode error reply: %v", err)
				return
			}
			fmt.Fprintf(w, "worker replied with error: [%d] [%d] %s",
				msgerr.CodeInfo[0], msgerr.CodeInfo[1], msgerr.Message)
			return
		case close:
			// close type
			return
		default:
			// protocol error
			log.Printf("protocol error: unknown message %v", msg)
		}
	}
}

func enqueueTask(id uint64, task []byte, output io.Writer) {
	encoder := codec.NewEncoder(output, hAsocket)
	// make http tunable
	encoder.Encode([]interface{}{id, invoke, []interface{}{"http"}})
	encoder.Encode([]interface{}{id, chunk, []interface{}{task}})
	encoder.Encode([]interface{}{id, close, []interface{}{}})
}
