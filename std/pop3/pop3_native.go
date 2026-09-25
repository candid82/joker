package pop3

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	. "github.com/candid82/joker/core"
)

type (
	POP3Client struct {
		client *pop3Client
		hash   uint32
	}

	pop3Client struct {
		conn    net.Conn
		reader  *bufio.Reader
		writer  *bufio.Writer
		timeout time.Duration
		mu      sync.Mutex
		closed  bool
	}

	connectOptions struct {
		addr       string
		username   string
		password   string
		tlsMode    string
		serverName string
		timeout    time.Duration
	}

	serverError struct {
		response string
	}
)

var pop3ClientType *Type

var makeTLSConfig = func(serverName string) *tls.Config {
	return &tls.Config{ServerName: serverName}
}

func MakePOP3Client(client *pop3Client) POP3Client {
	res := POP3Client{client: client}
	res.hash = HashPtr(uintptr(unsafe.Pointer(client)))
	return res
}

func (client POP3Client) ToString(_escape bool) string {
	return "#object[POP3Client]"
}

func (client POP3Client) Equals(other interface{}) bool {
	if otherClient, ok := other.(POP3Client); ok {
		return client.client == otherClient.client
	}
	return false
}

func (client POP3Client) GetInfo() *ObjectInfo {
	return nil
}

func (client POP3Client) GetType() *Type {
	return pop3ClientType
}

func (client POP3Client) Hash() uint32 {
	return client.hash
}

func (client POP3Client) WithInfo(_info *ObjectInfo) Object {
	return client
}

func EnsureArgIsPOP3Client(args []Object, index int) POP3Client {
	obj := args[index]
	if client, ok := obj.(POP3Client); ok {
		return client
	}
	panic(FailArg(obj, "POP3Client", index))
}

func ExtractPOP3Client(args []Object, index int) *pop3Client {
	return EnsureArgIsPOP3Client(args, index).client
}

func (err *serverError) Error() string {
	return err.response
}

func pop3Required(opts Map, key string) Object {
	if ok, value := opts.Get(MakeKeyword(key)); ok {
		return value
	}
	panic(RT.NewError(fmt.Sprintf(":%s key must be present in POP3 options map", key)))
}

func optionName(obj Object, field string) string {
	switch value := obj.(type) {
	case Keyword:
		return value.ToString(false)[1:]
	case Symbol:
		return value.ToString(false)
	case String:
		return value.S
	default:
		panic(RT.NewError(fmt.Sprintf("%s must be a keyword, symbol or string, got %s", field, obj.GetType().ToString(false))))
	}
}

func addressHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	return addr
}

func rejectNewlines(value string, field string) {
	if strings.ContainsAny(value, "\r\n") {
		panic(RT.NewError(field + " must not contain CR or LF"))
	}
}

func parseConnectOptions(opts Map) connectOptions {
	res := connectOptions{
		addr:     EnsureObjectIsString(pop3Required(opts, "addr"), "addr: %s").S,
		username: EnsureObjectIsString(pop3Required(opts, "username"), "username: %s").S,
		password: EnsureObjectIsString(pop3Required(opts, "password"), "password: %s").S,
		tlsMode:  "implicit",
	}
	rejectNewlines(res.username, "username")
	rejectNewlines(res.password, "password")
	if ok, value := opts.Get(MakeKeyword("tls")); ok {
		res.tlsMode = optionName(value, "tls")
	}
	switch res.tlsMode {
	case "implicit", "starttls", "none":
	default:
		panic(RT.NewError(":tls must be one of :implicit, :starttls or :none"))
	}
	if ok, value := opts.Get(MakeKeyword("server-name")); ok {
		res.serverName = EnsureObjectIsString(value, "server-name: %s").S
	} else {
		res.serverName = addressHost(res.addr)
	}
	if ok, value := opts.Get(MakeKeyword("timeout-ms")); ok {
		timeout := EnsureObjectIsInt(value, "timeout-ms: %s").I
		if timeout <= 0 {
			panic(RT.NewError(":timeout-ms must be positive"))
		}
		res.timeout = time.Duration(timeout) * time.Millisecond
	}
	return res
}

func newClient(conn net.Conn, timeout time.Duration) *pop3Client {
	return &pop3Client{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		writer:  bufio.NewWriter(conn),
		timeout: timeout,
	}
}

func (client *pop3Client) useConn(conn net.Conn) {
	client.conn = conn
	client.reader = bufio.NewReader(conn)
	client.writer = bufio.NewWriter(conn)
}

func (client *pop3Client) setDeadline() error {
	if client.timeout == 0 {
		return client.conn.SetDeadline(time.Time{})
	}
	return client.conn.SetDeadline(time.Now().Add(client.timeout))
}

func (client *pop3Client) clearDeadline() {
	_ = client.conn.SetDeadline(time.Time{})
}

func (client *pop3Client) closeLocked() error {
	if client.closed {
		return nil
	}
	client.closed = true
	return client.conn.Close()
}

func (client *pop3Client) readLine() (string, error) {
	line, err := client.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(line, "\r\n") {
		return "", errors.New("POP3 response line does not end with CRLF")
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func (client *pop3Client) readStatus() (string, error) {
	line, err := client.readLine()
	if err != nil {
		return "", err
	}
	switch {
	case line == "+OK":
		return "", nil
	case strings.HasPrefix(line, "+OK "):
		return line[4:], nil
	case line == "-ERR", strings.HasPrefix(line, "-ERR "):
		return "", &serverError{response: line}
	default:
		return "", fmt.Errorf("invalid POP3 status response: %q", line)
	}
}

func (client *pop3Client) writeCommand(command string) error {
	if _, err := client.writer.WriteString(command + "\r\n"); err != nil {
		return err
	}
	return client.writer.Flush()
}

func (client *pop3Client) command(command string) (string, error) {
	if err := client.writeCommand(command); err != nil {
		return "", err
	}
	return client.readStatus()
}

func (client *pop3Client) multiline(command string) ([]string, error) {
	if _, err := client.command(command); err != nil {
		return nil, err
	}
	var result []string
	for {
		line, err := client.readLine()
		if err != nil {
			return nil, err
		}
		if line == "." {
			return result, nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		result = append(result, line)
	}
}

func (client *pop3Client) multilineRaw(command string) (string, error) {
	if _, err := client.command(command); err != nil {
		return "", err
	}
	var result strings.Builder
	for {
		line, err := client.readLine()
		if err != nil {
			return "", err
		}
		if line == "." {
			return result.String(), nil
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		result.WriteString(line)
		result.WriteString("\r\n")
	}
}

func (client *pop3Client) execute(fn func() (interface{}, error)) (interface{}, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed {
		return nil, errors.New("POP3 client is closed")
	}
	if err := client.setDeadline(); err != nil {
		_ = client.closeLocked()
		return nil, err
	}
	defer client.clearDeadline()
	result, err := fn()
	if err != nil {
		var rejected *serverError
		if !errors.As(err, &rejected) {
			_ = client.closeLocked()
		}
	}
	return result, err
}

func connectNative(opts connectOptions) (*pop3Client, error) {
	dialer := &net.Dialer{Timeout: opts.timeout}
	conn, err := dialer.Dial("tcp", opts.addr)
	if err != nil {
		return nil, err
	}
	client := newClient(conn, opts.timeout)
	failed := true
	defer func() {
		if failed {
			_ = client.closeLocked()
		}
	}()
	run := func(fn func() error) error {
		if err := client.setDeadline(); err != nil {
			return err
		}
		defer client.clearDeadline()
		return fn()
	}
	if opts.tlsMode == "implicit" {
		tlsConn := tls.Client(conn, makeTLSConfig(opts.serverName))
		if err := run(tlsConn.Handshake); err != nil {
			return nil, fmt.Errorf("TLS handshake: %w", err)
		}
		client.useConn(tlsConn)
	}
	if err := run(func() error {
		_, err := client.readStatus()
		return err
	}); err != nil {
		return nil, fmt.Errorf("greeting: %w", err)
	}
	if opts.tlsMode == "starttls" {
		if err := run(func() error {
			_, err := client.command("STLS")
			return err
		}); err != nil {
			return nil, fmt.Errorf("STLS: %w", err)
		}
		tlsConn := tls.Client(conn, makeTLSConfig(opts.serverName))
		if err := run(tlsConn.Handshake); err != nil {
			return nil, fmt.Errorf("STLS handshake: %w", err)
		}
		client.useConn(tlsConn)
	}
	if err := run(func() error {
		_, err := client.command("USER " + opts.username)
		return err
	}); err != nil {
		return nil, fmt.Errorf("USER: %w", err)
	}
	if err := run(func() error {
		_, err := client.command("PASS " + opts.password)
		return err
	}); err != nil {
		return nil, fmt.Errorf("PASS: %w", err)
	}
	failed = false
	return client, nil
}

func runOperation(client *pop3Client, operation string, fn func() (interface{}, error)) interface{} {
	RT.GIL.Unlock()
	result, err := client.execute(fn)
	RT.GIL.Lock()
	if err != nil {
		panic(RT.NewError(fmt.Sprintf("POP3 %s: %s", operation, err)))
	}
	return result
}

func connect(opts Map) *pop3Client {
	options := parseConnectOptions(opts)
	RT.GIL.Unlock()
	client, err := connectNative(options)
	RT.GIL.Lock()
	if err != nil {
		panic(RT.NewError("POP3 connect: " + err.Error()))
	}
	return client
}

func parseNumber(value string, field string, allowZero bool) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 0 || (!allowZero && number == 0) {
		return 0, fmt.Errorf("invalid %s: %q", field, value)
	}
	return number, nil
}

func requireMessageNumber(number int) {
	if number <= 0 {
		panic(RT.NewError("POP3 message number must be positive"))
	}
}

func sizeMap(number int, octets int) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("number"), MakeInt(number))
	res.Add(MakeKeyword("octets"), MakeInt(octets))
	return res
}

func parseSize(line string) (int, int, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("invalid LIST response: %q", line)
	}
	number, err := parseNumber(fields[0], "message number", false)
	if err != nil {
		return 0, 0, err
	}
	octets, err := parseNumber(fields[1], "message size", true)
	return number, octets, err
}

func uidMap(number int, uid string) Map {
	res := EmptyArrayMap()
	res.Add(MakeKeyword("number"), MakeInt(number))
	res.Add(MakeKeyword("uid"), MakeString(uid))
	return res
}

func parseUID(line string) (int, string, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return 0, "", fmt.Errorf("invalid UIDL response: %q", line)
	}
	number, err := parseNumber(fields[0], "message number", false)
	if err != nil {
		return 0, "", err
	}
	return number, fields[1], nil
}

func capabilities(client *pop3Client) Map {
	lines := runOperation(client, "CAPA", func() (interface{}, error) {
		return client.multiline("CAPA")
	}).([]string)
	result := EmptyArrayMap()
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			panic(RT.NewError("POP3 CAPA: invalid empty capability response"))
		}
		args := make([]string, len(fields)-1)
		copy(args, fields[1:])
		result.Add(MakeString(fields[0]), MakeStringVector(args))
	}
	return result
}

func stat(client *pop3Client) Map {
	line := runOperation(client, "STAT", func() (interface{}, error) {
		return client.command("STAT")
	}).(string)
	fields := strings.Fields(line)
	if len(fields) < 2 {
		panic(RT.NewError(fmt.Sprintf("POP3 STAT: invalid response: %q", line)))
	}
	count, err := parseNumber(fields[0], "message count", true)
	if err != nil {
		panic(RT.NewError("POP3 STAT: " + err.Error()))
	}
	octets, err := parseNumber(fields[1], "mailbox size", true)
	if err != nil {
		panic(RT.NewError("POP3 STAT: " + err.Error()))
	}
	result := EmptyArrayMap()
	result.Add(MakeKeyword("count"), MakeInt(count))
	result.Add(MakeKeyword("octets"), MakeInt(octets))
	return result
}

func listAll(client *pop3Client) Object {
	lines := runOperation(client, "LIST", func() (interface{}, error) {
		return client.multiline("LIST")
	}).([]string)
	result := EmptyArrayVector()
	for _, line := range lines {
		number, octets, err := parseSize(line)
		if err != nil {
			panic(RT.NewError("POP3 LIST: " + err.Error()))
		}
		result.Append(sizeMap(number, octets))
	}
	return result
}

func listOne(client *pop3Client, messageNumber int) Object {
	requireMessageNumber(messageNumber)
	line := runOperation(client, "LIST", func() (interface{}, error) {
		return client.command(fmt.Sprintf("LIST %d", messageNumber))
	}).(string)
	number, octets, err := parseSize(line)
	if err != nil {
		panic(RT.NewError("POP3 LIST: " + err.Error()))
	}
	if number != messageNumber {
		panic(RT.NewError(fmt.Sprintf("POP3 LIST: response identifies message %d, requested %d", number, messageNumber)))
	}
	return sizeMap(number, octets)
}

func uidlAll(client *pop3Client) Object {
	lines := runOperation(client, "UIDL", func() (interface{}, error) {
		return client.multiline("UIDL")
	}).([]string)
	result := EmptyArrayVector()
	for _, line := range lines {
		number, uid, err := parseUID(line)
		if err != nil {
			panic(RT.NewError("POP3 UIDL: " + err.Error()))
		}
		result.Append(uidMap(number, uid))
	}
	return result
}

func uidlOne(client *pop3Client, messageNumber int) Object {
	requireMessageNumber(messageNumber)
	line := runOperation(client, "UIDL", func() (interface{}, error) {
		return client.command(fmt.Sprintf("UIDL %d", messageNumber))
	}).(string)
	number, uid, err := parseUID(line)
	if err != nil {
		panic(RT.NewError("POP3 UIDL: " + err.Error()))
	}
	if number != messageNumber {
		panic(RT.NewError(fmt.Sprintf("POP3 UIDL: response identifies message %d, requested %d", number, messageNumber)))
	}
	return uidMap(number, uid)
}

func retrieve(client *pop3Client, messageNumber int) string {
	requireMessageNumber(messageNumber)
	return runOperation(client, "RETR", func() (interface{}, error) {
		return client.multilineRaw(fmt.Sprintf("RETR %d", messageNumber))
	}).(string)
}

func top(client *pop3Client, messageNumber int, lines int) string {
	requireMessageNumber(messageNumber)
	if lines < 0 {
		panic(RT.NewError("POP3 TOP line count must be non-negative"))
	}
	return runOperation(client, "TOP", func() (interface{}, error) {
		return client.multilineRaw(fmt.Sprintf("TOP %d %d", messageNumber, lines))
	}).(string)
}

func successCommand(client *pop3Client, operation string, command string) Nil {
	runOperation(client, operation, func() (interface{}, error) {
		_, err := client.command(command)
		return nil, err
	})
	return NIL
}

func deleteMessage(client *pop3Client, messageNumber int) Nil {
	requireMessageNumber(messageNumber)
	return successCommand(client, "DELE", fmt.Sprintf("DELE %d", messageNumber))
}

func reset(client *pop3Client) Nil {
	return successCommand(client, "RSET", "RSET")
}

func noop(client *pop3Client) Nil {
	return successCommand(client, "NOOP", "NOOP")
}

func quit(client *pop3Client) Nil {
	RT.GIL.Unlock()
	client.mu.Lock()
	var err error
	if client.closed {
		err = errors.New("POP3 client is closed")
	} else {
		if deadlineErr := client.setDeadline(); deadlineErr != nil {
			err = deadlineErr
		} else {
			_, err = client.command("QUIT")
			client.clearDeadline()
		}
		closeErr := client.closeLocked()
		if err == nil {
			err = closeErr
		}
	}
	client.mu.Unlock()
	RT.GIL.Lock()
	if err != nil {
		panic(RT.NewError("POP3 QUIT: " + err.Error()))
	}
	return NIL
}

func close(client *pop3Client) Nil {
	RT.GIL.Unlock()
	client.mu.Lock()
	err := client.closeLocked()
	client.mu.Unlock()
	RT.GIL.Lock()
	if err != nil {
		panic(RT.NewError("POP3 close: " + err.Error()))
	}
	return NIL
}

func init() {
	pop3ClientType = RegType("POP3Client", (*POP3Client)(nil), "Wraps an authenticated POP3 client session")
}
