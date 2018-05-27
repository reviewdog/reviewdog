package cookieman

import (
	"encoding/base64"
	"net/http"
)

type Cipher interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

type CookieMan struct {
	defaultOpt CookieOption
	cipher     Cipher
}

type CookieOption struct {
	http.Cookie
}

type CookieStore struct {
	name      string
	cookieman *CookieMan
	opt       *CookieOption
}

func (cs *CookieStore) Name() string {
	return cs.name
}

func (cs *CookieStore) Set(w http.ResponseWriter, value []byte) error {
	return cs.cookieman.Set(w, cs.name, value, cs.opt)
}

func (cs *CookieStore) Get(r *http.Request) ([]byte, error) {
	return cs.cookieman.Get(r, cs.name)
}

func (cs *CookieStore) Clear(w http.ResponseWriter) {
	cs.cookieman.Clear(w, cs.name)
}

func New(cipher Cipher, defaultOpt CookieOption) *CookieMan {
	return &CookieMan{defaultOpt: defaultOpt, cipher: cipher}
}

func (c *CookieMan) NewCookieStore(name string, opt *CookieOption) *CookieStore {
	return &CookieStore{
		name:      name,
		cookieman: c,
		opt:       opt,
	}
}

func (c *CookieMan) Set(w http.ResponseWriter, name string, value []byte, opt *CookieOption) error {
	v, err := c.cipher.Encrypt(value)
	if err != nil {
		return err
	}
	http.SetCookie(w, c.cookie(name, base64.URLEncoding.EncodeToString(v), opt))
	return nil
}

func (c *CookieMan) Get(r *http.Request, name string) ([]byte, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.URLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, err
	}
	return c.cipher.Decrypt(ciphertext)
}

func (c *CookieMan) Clear(w http.ResponseWriter, name string) {
	opt := &CookieOption{}
	opt.MaxAge = -1
	http.SetCookie(w, c.cookie(name, "", opt))
}

func (c *CookieMan) cookie(name, value string, opt *CookieOption) *http.Cookie {
	cookie := c.defaultOpt.Cookie
	cookie.Name = name
	cookie.Value = value
	if opt == nil {
		return &cookie
	}
	if opt.Path != "" {
		cookie.Path = opt.Path
	}
	if opt.Domain != "" {
		cookie.Domain = opt.Domain
	}
	if opt.MaxAge != 0 {
		cookie.MaxAge = opt.MaxAge
	}
	if !opt.Expires.IsZero() {
		cookie.Expires = opt.Expires
	}
	if opt.Secure {
		cookie.Secure = opt.Secure
	}
	if opt.HttpOnly {
		cookie.HttpOnly = opt.HttpOnly
	}
	return &cookie
}
