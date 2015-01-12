package session

import (
	"crypto/aes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type User struct {
	Username string
	NickName string
}

func TestGenerate(t *testing.T) {
	str := generateRandomKey(20)
	if len(str) != 20 {
		t.Fatal("generate length is not equal to 20")
	}
}

func TestCookieEncodeDecode(t *testing.T) {
	hashKey := "testhashKey"
	blockkey := generateRandomKey(16)
	block, err := aes.NewCipher(blockkey)
	if err != nil {
		t.Fatal("NewCipher:", err)
	}
	securityName := string(generateRandomKey(20))
	val := make(map[interface{}]interface{})
	val["tag"] = "hello"
	val["color"] = "blue"
	str, err := encodeCookie(block, hashKey, securityName, val)
	if err != nil {
		t.Fatal("encodeCookie:", err)
	}
	dst := make(map[interface{}]interface{})
	dst, err = decodeCookie(block, hashKey, securityName, str, 3600)
	if err != nil {
		t.Fatal("decodeCookie", err)
	}
	if dst["tag"] != "hello" {
		t.Fatal("dst get map error")
	}
	if dst["color"] != "blue" {
		t.Fatal("dst get map error")
	}
}

func TestParseConfig(t *testing.T) {
	s := `{"cookieName":"gosessionid","gclifetime":3600}`
	cf := new(managerConfig)
	cf.EnableSetCookie = true
	err := json.Unmarshal([]byte(s), cf)
	if err != nil {
		t.Fatal("parse json error,", err)
	}
	if cf.CookieName != "gosessionid" {
		t.Fatal("parseconfig get cookiename error")
	}
	if cf.Gclifetime != 3600 {
		t.Fatal("parseconfig get gclifetime error")
	}

	cc := `{"cookieName":"gosessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"gosessionid\",\"securityKey\":\"flotillacookiehashkey\"}"}`
	cf2 := new(managerConfig)
	cf2.EnableSetCookie = true
	err = json.Unmarshal([]byte(cc), cf2)
	if err != nil {
		t.Fatal("parse json error,", err)
	}
	if cf2.CookieName != "gosessionid" {
		t.Fatal("parseconfig get cookiename error")
	}
	if cf2.Gclifetime != 3600 {
		t.Fatal("parseconfig get gclifetime error")
	}
	if cf2.EnableSetCookie != false {
		t.Fatal("parseconfig get enableSetCookie error")
	}
	cconfig := new(cookieConfig)
	err = json.Unmarshal([]byte(cf2.ProviderConfig), cconfig)
	if err != nil {
		t.Fatal("parse ProviderConfig err,", err)
	}
	if cconfig.CookieName != "gosessionid" {
		t.Fatal("ProviderConfig get cookieName error")
	}
	if cconfig.SecurityKey != "flotillacookiehashkey" {
		t.Fatal("ProviderConfig get securityKey error")
	}
}

func TestCookie(t *testing.T) {
	config := `{"cookieName":"gosessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"gosessionid\",\"securityKey\":\"flotillacookiehashkey\"}"}`
	globalSessions, err := NewManager("cookie", config)
	if err != nil {
		t.Fatal("init cookie session err", err)
	}
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	//sess := globalSessions.SessionStart(w, r)
	sess, err := globalSessions.SessionStart(w, r)
	if err != nil {
		t.Fatal("set error,", err)
	}
	err = sess.Set("username", "Special Agent Dana Scully")
	if err != nil {
		t.Fatal("set error,", err)
	}
	if username := sess.Get("username"); username != "Special Agent Dana Scully" {
		t.Fatal("get username error")
	}
	sess.SessionRelease(w)
	if cookiestr := w.Header().Get("Set-Cookie"); cookiestr == "" {
		t.Fatal("setcookie error")
	} else {
		parts := strings.Split(strings.TrimSpace(cookiestr), ";")
		for k, v := range parts {
			nameval := strings.Split(v, "=")
			if k == 0 && nameval[0] != "gosessionid" {
				t.Fatal("error")
			}
		}
	}
}
