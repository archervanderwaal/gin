package gin

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sort"
)

type (
	BasicAuthPair struct {
		Code string
		User string
	}
	Account struct {
		User     string
		Password string
	}

	Accounts []Account
	Pairs    []BasicAuthPair
)

func (a Pairs) Len() int           { return len(a) }
func (a Pairs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Pairs) Less(i, j int) bool { return a[i].Code < a[j].Code }

// 处理访问凭证, 对访问凭证进行编码, 返回pairs
func processCredentials(accounts Accounts) (Pairs, error) {
	if len(accounts) == 0 {
		return nil, errors.New("Empty list of authorized credentials.")
	}
	pairs := make(Pairs, 0, len(accounts))
	for _, account := range accounts {
		if len(account.User) == 0 || len(account.Password) == 0 {
			return nil, errors.New("User or password is empty")
		}
		base := account.User + ":" + account.Password
		// base64编码
		code := "Basic " + base64.StdEncoding.EncodeToString([]byte(base))
		pairs = append(pairs, BasicAuthPair{code, account.User})
	}
	// We have to sort the credentials in order to use bsearch later.
	// 排序方便后面二分查找
	sort.Sort(pairs)
	return pairs, nil
}

// 查询凭证
func searchCredential(pairs Pairs, auth string) string {
	if len(auth) == 0 {
		return ""
	}
	// 在有效访问凭证中查询授权用户
	// Search user in the slice of allowed credentials
	r := sort.Search(len(pairs), func(i int) bool { return pairs[i].Code >= auth })

	// 判断是否有效
	if r < len(pairs) && subtle.ConstantTimeCompare([]byte(pairs[r].Code), []byte(auth)) == 1 {
		// user is allowed, set UserId to key "user" in this context, the userId can be read later using
		// c.Get("user"
		return pairs[r].User
	} else {
		return ""
	}
}

// Implements a basic Basic HTTP Authorization. It takes as argument a map[string]string where
// the key is the user name and the value is the password.
func BasicAuth(accounts Accounts) HandlerFunc {

	pairs, err := processCredentials(accounts)
	if err != nil {
		panic(err)
	}
	return func(c *Context) {
		// Search user in the slice of allowed credentials
		user := searchCredential(pairs, c.Req.Header.Get("Authorization"))
		if len(user) == 0 {
			// Credentials doesn't match, we return 401 Unauthorized and abort request.
			c.Writer.Header().Set("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
			c.Fail(401, errors.New("Unauthorized"))
		} else {
			// user is allowed, set UserId to key "user" in this context, the userId can be read later using
			// c.Get("user")
			c.Set("user", user)
		}
	}
}
