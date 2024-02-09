/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"net"
	"net/url"
	"strings"

	"github.com/go-sql-driver/mysql"
)

/*
MySQL Go driver (https://github.com/go-sql-driver/mysql) uses a custom connection string (DSN) format.
[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
*/

func (a AppBinding) MySQLDSN() (string, error) {
	dsn, err := a.URL()
	if err != nil {
		return "", err
	}
	return CanonicalMySQLDSN(dsn)
}

// CanonicalMySQLDSN will convert a regular URL into MySQL DSN format
func CanonicalMySQLDSN(dsn string) (string, error) {
	_, err := mysql.ParseDSN(dsn)
	if err == nil {
		return dsn, nil
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}

	rebuild := mysql.NewConfig()
	rebuild.Net = u.Scheme
	rebuild.Addr = u.Host
	rebuild.DBName = strings.TrimPrefix(u.Path, "/")
	if u.User != nil {
		rebuild.User = u.User.Username()
		if pass, found := u.User.Password(); found {
			rebuild.Passwd = pass
		}
	}
	rebuild.Params = map[string]string{}
	for k, v := range u.Query() {
		rebuild.Params[k] = v[0]
	}
	return rebuild.FormatDSN(), nil
}

func ParseMySQLHost(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	return cfg.Addr, err
}

func ParseMySQLHostname(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	return host, err
}

func ParseMySQLPort(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	_, port, err := net.SplitHostPort(cfg.Addr)
	return port, err
}
