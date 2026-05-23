package smtp

import (
	"fmt"
	"net"
	netsmtp "net/smtp"
	"strings"

	. "github.com/candid82/joker/core"
)

func requiredIn(m Map, key string, context string) Object {
	k := MakeKeyword(key)
	if ok, v := m.Get(k); ok {
		return v
	}
	panic(RT.NewError(fmt.Sprintf(":%s key must be present in %s map", key, context)))
}

func required(m Map, key string) Object {
	return requiredIn(m, key, "SMTP request")
}

func keywordName(obj Object, field string) string {
	switch obj := obj.(type) {
	case Keyword:
		return obj.ToString(false)[1:]
	case Symbol:
		return obj.ToString(false)
	case String:
		return obj.S
	default:
		panic(RT.NewError(fmt.Sprintf("%s must be a keyword, symbol or string, got %s", field, obj.GetType().ToString(false))))
	}
}

func stringSeq(obj Object, field string) []string {
	seq := EnsureObjectIsSeqable(obj, field+": %s").Seq()
	res := make([]string, 0)
	for !seq.IsEmpty() {
		res = append(res, EnsureObjectIsString(seq.First(), field+": %s").S)
		seq = seq.Rest()
	}
	return res
}

func smtpHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		return strings.TrimPrefix(strings.TrimSuffix(addr, "]"), "[")
	}
	return addr
}

func authFromMap(addr string, authMap Map) netsmtp.Auth {
	typeObj := requiredIn(authMap, "type", "SMTP auth")
	switch typ := keywordName(typeObj, "auth type"); typ {
	case "plain":
		identity := ""
		if ok, value := authMap.Get(MakeKeyword("identity")); ok {
			identity = EnsureObjectIsString(value, "auth identity: %s").S
		}
		host := smtpHost(addr)
		if ok, value := authMap.Get(MakeKeyword("host")); ok {
			host = EnsureObjectIsString(value, "auth host: %s").S
		}
		username := EnsureObjectIsString(requiredIn(authMap, "username", "SMTP auth"), "auth username: %s").S
		password := EnsureObjectIsString(requiredIn(authMap, "password", "SMTP auth"), "auth password: %s").S
		return netsmtp.PlainAuth(identity, username, password, host)
	case "cram-md5":
		username := EnsureObjectIsString(requiredIn(authMap, "username", "SMTP auth"), "auth username: %s").S
		secret := EnsureObjectIsString(requiredIn(authMap, "secret", "SMTP auth"), "auth secret: %s").S
		return netsmtp.CRAMMD5Auth(username, secret)
	default:
		panic(RT.NewError(fmt.Sprintf("unsupported SMTP auth type: %s", typ)))
	}
}

func sendMessage(request Map) Object {
	addr := EnsureObjectIsString(required(request, "addr"), "addr: %s").S
	from := EnsureObjectIsString(required(request, "from"), "from: %s").S
	to := stringSeq(required(request, "to"), "to")
	if len(to) == 0 {
		panic(RT.NewError(":to must contain at least one recipient"))
	}
	message := EnsureObjectIsString(required(request, "message"), "message: %s").S

	var auth netsmtp.Auth
	if ok, value := request.Get(MakeKeyword("auth")); ok && value != NIL {
		auth = authFromMap(addr, EnsureObjectIsMap(value, "auth: %s"))
	}

	RT.GIL.Unlock()
	err := netsmtp.SendMail(addr, auth, from, to, []byte(message))
	RT.GIL.Lock()
	PanicOnErr(err)
	return NIL
}
