package transports

import (
	"github.com/aacfactory/fns/commons/objects"
	"sync"
	"time"
)

const (
	CookieSameSiteDisabled CookieSameSite = iota
	CookieSameSiteDefaultMode
	CookieSameSiteLaxMode
	CookieSameSiteStrictMode
	CookieSameSiteNoneMode
)

type CookieSameSite int

type Cookie struct {
	noCopy   objects.NoCopy
	key      []byte
	value    []byte
	expire   time.Time
	maxAge   int
	domain   []byte
	path     []byte
	httpOnly bool
	secure   bool
	sameSite CookieSameSite
}

func (c *Cookie) HTTPOnly() bool {
	return c.httpOnly
}

func (c *Cookie) SetHTTPOnly(httpOnly bool) {
	c.httpOnly = httpOnly
}

func (c *Cookie) Secure() bool {
	return c.secure
}

func (c *Cookie) SetSecure(secure bool) {
	c.secure = secure
}

func (c *Cookie) SameSite() CookieSameSite {
	return c.sameSite
}

func (c *Cookie) SetSameSite(mode CookieSameSite) {
	c.sameSite = mode
	if mode == CookieSameSiteNoneMode {
		c.SetSecure(true)
	}
}

func (c *Cookie) Path() []byte {
	return c.path
}

func (c *Cookie) SetPath(path []byte) {
	c.path = path
}

func (c *Cookie) Domain() []byte {
	return c.domain
}

func (c *Cookie) SetDomain(domain []byte) {
	c.domain = append(c.domain[:0], domain...)
}

func (c *Cookie) MaxAge() int {
	return c.maxAge
}

func (c *Cookie) SetMaxAge(seconds int) {
	c.maxAge = seconds
}

func (c *Cookie) Expire() time.Time {
	return c.expire
}

func (c *Cookie) SetExpire(expire time.Time) {
	c.expire = expire
}

func (c *Cookie) Value() []byte {
	return c.value
}

func (c *Cookie) SetValue(value []byte) {
	c.value = append(c.value[:0], value...)
}

func (c *Cookie) Key() []byte {
	return c.key
}

func (c *Cookie) SetKey(key []byte) {
	c.key = append(c.key[:0], key...)
}

func (c *Cookie) Reset() {
	c.key = c.key[:0]
	c.value = c.value[:0]
	c.expire = time.Time{}
	c.maxAge = 0
	c.domain = c.domain[:0]
	c.path = c.path[:0]
	c.httpOnly = false
	c.secure = false
	c.sameSite = CookieSameSiteDisabled
}

var cookiePool = &sync.Pool{
	New: func() interface{} {
		return &Cookie{}
	},
}

func AcquireCookie() *Cookie {
	return cookiePool.Get().(*Cookie)
}

func ReleaseCookie(c *Cookie) {
	c.Reset()
	cookiePool.Put(c)
}
