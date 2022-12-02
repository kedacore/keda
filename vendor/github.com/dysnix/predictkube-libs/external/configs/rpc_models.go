package configs

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/xhit/go-str2duration/v2"
)

type Cache struct {
	GlobalTTL TTL     `yaml:",inline"` //`yaml:"globalTtl" json:"global_ttl"`
	Redis     *Redis  `yaml:"redis,omitempty" json:"redis,omitempty"`
	Memory    *Memory `yaml:"memory,omitempty" json:"memory,omitempty"`
}

type TTL struct {
	TTL time.Duration
}

func (c *TTL) MarshalYAML() (interface{}, error) {
	type alias struct {
		TTL string
	}

	if c == nil {
		*c = TTL{}
	}

	return alias{
		TTL: HumanDuration(c.TTL),
	}, nil
}

func (c *TTL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		TTL string
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if c == nil {
		*c = TTL{}
	}

	c.TTL, err = str2duration.ParseDuration(tmp.TTL)
	if err != nil {
		return err
	}

	return nil
}

func (c *TTL) MarshalJSON() ([]byte, error) {
	type alias struct {
		TTL string
	}

	if c == nil {
		*c = TTL{}
	}

	return json.Marshal(alias{
		TTL: HumanDuration(c.TTL),
	})
}

func (c *TTL) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		TTL string
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if c == nil {
		*c = TTL{}
	}

	c.TTL, err = str2duration.ParseDuration(tmp.TTL)
	if err != nil {
		return err
	}

	return nil
}

type Memory struct {
	CleanupInterval time.Duration `yaml:"cleanupInterval" json:"cleanup_interval"`
}

func (m *Memory) MarshalYAML() (interface{}, error) {
	type alias struct {
		CleanupInterval string `yaml:"cleanupInterval" json:"cleanup_interval"`
	}

	if m == nil {
		*m = Memory{}
	}

	return alias{
		CleanupInterval: HumanDuration(m.CleanupInterval),
	}, nil
}

func (m *Memory) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		CleanupInterval string `yaml:"cleanupInterval" json:"cleanup_interval"`
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if m == nil {
		*m = Memory{}
	}

	m.CleanupInterval, err = str2duration.ParseDuration(tmp.CleanupInterval)
	if err != nil {
		return err
	}

	return nil
}

func (m *Memory) MarshalJSON() ([]byte, error) {
	type alias struct {
		CleanupInterval string `yaml:"cleanupInterval" json:"cleanup_interval"`
	}

	if m == nil {
		*m = Memory{}
	}

	return json.Marshal(alias{
		CleanupInterval: HumanDuration(m.CleanupInterval),
	})
}

func (m *Memory) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		CleanupInterval string `yaml:"cleanupInterval" json:"cleanup_interval"`
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if m == nil {
		*m = Memory{}
	}

	m.CleanupInterval, err = str2duration.ParseDuration(tmp.CleanupInterval)
	if err != nil {
		return err
	}

	return nil
}

type Redis struct {
	Username        string            `validate:"ascii"`
	Password        string            `validate:"ascii"`
	DB              uint              `yaml:"database" json:"database" validate:"omitempty"`
	RequestTimeout  time.Duration     `yaml:"requestTimeout" json:"request_timeout"`
	MaxRedirects    int               `yaml:"maxRedirects" json:"max_redirects" validate:"numeric"`
	ReadAddrs       []string          `yaml:"readAddrs" json:"read_addrs" validate:"required"`
	WriteAddrs      []string          `yaml:"writeAddrs" json:"write_addrs" validate:"required"`
	MinRetryBackoff time.Duration     `yaml:"minRetryBackoff" json:"min_retry_backoff" validate:"required"`
	MaxRetryBackoff time.Duration     `yaml:"maxRetryBackoff" json:"max_retry_backoff" validate:"required"`
	DialTimeout     time.Duration     `yaml:"dialTimeout" json:"dial_timeout" validate:"required"`
	ReadTimeout     time.Duration     `yaml:"readTimeout" json:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration     `yaml:"writeTimeout" json:"write_timeout" validate:"required"`
	Pool            *RedisClusterPool `yaml:"pool" json:"pool" validate:"required"`
	MaxRetries      int               `yaml:"maxRetries" json:"max_retries" validate:"numeric"`
}

func (r *Redis) MarshalYAML() (interface{}, error) {
	type alias struct {
		Username        string            `validate:"ascii"`
		Password        string            `validate:"ascii"`
		RequestTimeout  string            `yaml:"requestTimeout" json:"request_timeout"`
		MaxRedirects    int               `yaml:"maxRedirects" json:"max_redirects" validate:"numeric"`
		ReadAddrs       string            `yaml:"readAddrs" json:"read_addrs" validate:"required"`
		WriteAddrs      string            `yaml:"writeAddrs" json:"write_addrs" validate:"required"`
		MinRetryBackoff string            `yaml:"minRetryBackoff" json:"min_retry_backoff" validate:"required"`
		MaxRetryBackoff string            `yaml:"maxRetryBackoff" json:"max_retry_backoff" validate:"required"`
		DialTimeout     string            `yaml:"dialTimeout" json:"dial_timeout" validate:"required"`
		ReadTimeout     string            `yaml:"readTimeout" json:"read_timeout" validate:"required"`
		WriteTimeout    string            `yaml:"writeTimeout" json:"write_timeout" validate:"required"`
		Pool            *RedisClusterPool `yaml:"pool" json:"pool" validate:"required"`
		MaxRetries      int               `yaml:"maxRetries" json:"max_retries" validate:"numeric"`
	}

	if r == nil {
		*r = Redis{}
	}

	return alias{
		Username:        r.Username,
		Password:        r.Password,
		RequestTimeout:  HumanDuration(r.RequestTimeout),
		MaxRedirects:    r.MaxRedirects,
		ReadAddrs:       strings.Join(r.ReadAddrs, ","),
		WriteAddrs:      strings.Join(r.WriteAddrs, ","),
		MinRetryBackoff: HumanDuration(r.MinRetryBackoff),
		MaxRetryBackoff: HumanDuration(r.MaxRetryBackoff),
		DialTimeout:     HumanDuration(r.DialTimeout),
		ReadTimeout:     HumanDuration(r.ReadTimeout),
		WriteTimeout:    HumanDuration(r.WriteTimeout),
		Pool:            r.Pool,
		MaxRetries:      r.MaxRetries,
	}, nil
}

func (r *Redis) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		Username        string            `validate:"ascii"`
		Password        string            `validate:"ascii"`
		RequestTimeout  string            `yaml:"requestTimeout" json:"request_timeout"`
		MaxRedirects    int               `yaml:"maxRedirects" json:"max_redirects" validate:"numeric"`
		ReadAddrs       string            `yaml:"readAddrs" json:"read_addrs" validate:"required"`
		WriteAddrs      string            `yaml:"writeAddrs" json:"write_addrs" validate:"required"`
		MinRetryBackoff string            `yaml:"minRetryBackoff" json:"min_retry_backoff" validate:"required"`
		MaxRetryBackoff string            `yaml:"maxRetryBackoff" json:"max_retry_backoff" validate:"required"`
		DialTimeout     string            `yaml:"dialTimeout" json:"dial_timeout" validate:"required"`
		ReadTimeout     string            `yaml:"readTimeout" json:"read_timeout" validate:"required"`
		WriteTimeout    string            `yaml:"writeTimeout" json:"write_timeout" validate:"required"`
		Pool            *RedisClusterPool `yaml:"pool" json:"pool" validate:"required"`
		MaxRetries      int               `yaml:"maxRetries" json:"max_retries" validate:"numeric"`
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if r == nil {
		*r = Redis{}
	}

	r.Username = tmp.Username
	r.Password = tmp.Password
	r.MaxRedirects = tmp.MaxRedirects
	r.Pool = tmp.Pool
	r.MaxRetries = tmp.MaxRetries

	r.ReadAddrs = strings.Split(tmp.ReadAddrs, ",")
	r.WriteAddrs = strings.Split(tmp.WriteAddrs, ",")

	r.RequestTimeout, err = str2duration.ParseDuration(tmp.RequestTimeout)
	if err != nil {
		return err
	}

	r.MinRetryBackoff, err = str2duration.ParseDuration(tmp.MinRetryBackoff)
	if err != nil {
		return err
	}

	r.MaxRetryBackoff, err = str2duration.ParseDuration(tmp.MaxRetryBackoff)
	if err != nil {
		return err
	}

	r.DialTimeout, err = str2duration.ParseDuration(tmp.DialTimeout)
	if err != nil {
		return err
	}

	r.ReadTimeout, err = str2duration.ParseDuration(tmp.ReadTimeout)
	if err != nil {
		return err
	}

	r.WriteTimeout, err = str2duration.ParseDuration(tmp.WriteTimeout)
	if err != nil {
		return err
	}

	return nil
}

func (r *Redis) MarshalJSON() ([]byte, error) {
	type alias struct {
		Username        string            `validate:"ascii"`
		Password        string            `validate:"ascii"`
		RequestTimeout  string            `yaml:"requestTimeout" json:"request_timeout"`
		MaxRedirects    int               `yaml:"maxRedirects" json:"max_redirects" validate:"numeric"`
		ReadAddrs       string            `yaml:"readAddrs" json:"read_addrs" validate:"required"`
		WriteAddrs      string            `yaml:"writeAddrs" json:"write_addrs" validate:"required"`
		MinRetryBackoff string            `yaml:"minRetryBackoff" json:"min_retry_backoff" validate:"required"`
		MaxRetryBackoff string            `yaml:"maxRetryBackoff" json:"max_retry_backoff" validate:"required"`
		DialTimeout     string            `yaml:"dialTimeout" json:"dial_timeout" validate:"required"`
		ReadTimeout     string            `yaml:"readTimeout" json:"read_timeout" validate:"required"`
		WriteTimeout    string            `yaml:"writeTimeout" json:"write_timeout" validate:"required"`
		Pool            *RedisClusterPool `yaml:"pool" json:"pool" validate:"required"`
		MaxRetries      int               `yaml:"maxRetries" json:"max_retries" validate:"numeric"`
	}

	if r == nil {
		*r = Redis{}
	}

	return json.Marshal(alias{
		Username:        r.Username,
		Password:        r.Password,
		RequestTimeout:  HumanDuration(r.RequestTimeout),
		MaxRedirects:    r.MaxRedirects,
		ReadAddrs:       strings.Join(r.ReadAddrs, ","),
		WriteAddrs:      strings.Join(r.WriteAddrs, ","),
		MinRetryBackoff: HumanDuration(r.MinRetryBackoff),
		MaxRetryBackoff: HumanDuration(r.MaxRetryBackoff),
		DialTimeout:     HumanDuration(r.DialTimeout),
		ReadTimeout:     HumanDuration(r.ReadTimeout),
		WriteTimeout:    HumanDuration(r.WriteTimeout),
		Pool:            r.Pool,
		MaxRetries:      r.MaxRetries,
	})
}

func (r *Redis) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		Username        string            `validate:"ascii"`
		Password        string            `validate:"ascii"`
		RequestTimeout  string            `yaml:"requestTimeout" json:"request_timeout"`
		MaxRedirects    int               `yaml:"maxRedirects" json:"max_redirects" validate:"numeric"`
		ReadAddrs       string            `yaml:"readAddrs" json:"read_addrs" validate:"required"`
		WriteAddrs      string            `yaml:"writeAddrs" json:"write_addrs" validate:"required"`
		MinRetryBackoff string            `yaml:"minRetryBackoff" json:"min_retry_backoff" validate:"required"`
		MaxRetryBackoff string            `yaml:"maxRetryBackoff" json:"max_retry_backoff" validate:"required"`
		DialTimeout     string            `yaml:"dialTimeout" json:"dial_timeout" validate:"required"`
		ReadTimeout     string            `yaml:"readTimeout" json:"read_timeout" validate:"required"`
		WriteTimeout    string            `yaml:"writeTimeout" json:"write_timeout" validate:"required"`
		Pool            *RedisClusterPool `yaml:"pool" json:"pool" validate:"required"`
		MaxRetries      int               `yaml:"maxRetries" json:"max_retries" validate:"numeric"`
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if r == nil {
		*r = Redis{}
	}

	r.Username = tmp.Username
	r.Password = tmp.Password
	r.MaxRedirects = tmp.MaxRedirects
	r.Pool = tmp.Pool
	r.MaxRetries = tmp.MaxRetries

	r.ReadAddrs = strings.Split(tmp.ReadAddrs, ",")
	r.WriteAddrs = strings.Split(tmp.WriteAddrs, ",")

	r.RequestTimeout, err = str2duration.ParseDuration(tmp.RequestTimeout)
	if err != nil {
		return err
	}

	r.MinRetryBackoff, err = str2duration.ParseDuration(tmp.MinRetryBackoff)
	if err != nil {
		return err
	}

	r.MaxRetryBackoff, err = str2duration.ParseDuration(tmp.MaxRetryBackoff)
	if err != nil {
		return err
	}

	r.DialTimeout, err = str2duration.ParseDuration(tmp.DialTimeout)
	if err != nil {
		return err
	}

	r.ReadTimeout, err = str2duration.ParseDuration(tmp.ReadTimeout)
	if err != nil {
		return err
	}

	r.WriteTimeout, err = str2duration.ParseDuration(tmp.WriteTimeout)
	if err != nil {
		return err
	}

	return nil
}

type RedisClusterPool struct {
	PoolSize           int           `yaml:"poolSize" json:"pool_size" validate:"numeric"`
	MinIdleConns       int           `yaml:"minIdleConns" json:"min_idle_conns" validate:"numeric"`
	MaxConnAge         time.Duration `yaml:"maxConnAge" json:"max_conn_age"`
	PoolTimeout        time.Duration `yaml:"poolTimeout" json:"pool_timeout"`
	IdleTimeout        time.Duration `yaml:"idleTimeout" json:"idle_timeout"`
	IdleCheckFrequency time.Duration `yaml:"idleCheckTimeout" json:"idle_check_timeout"`
}

func (rcp *RedisClusterPool) MarshalYAML() (interface{}, error) {
	type alias struct {
		PoolSize           int    `yaml:"poolSize" json:"pool_size" validate:"numeric"`
		MinIdleConns       int    `yaml:"minIdleConns" json:"min_idle_conns" validate:"numeric"`
		MaxConnAge         string `yaml:"maxConnAge" json:"max_conn_age"`
		PoolTimeout        string `yaml:"poolTimeout" json:"pool_timeout"`
		IdleTimeout        string `yaml:"idleTimeout" json:"idle_timeout"`
		IdleCheckFrequency string `yaml:"idleCheckTimeout" json:"idle_check_timeout"`
	}

	if rcp == nil {
		*rcp = RedisClusterPool{}
	}

	return alias{
		PoolSize:           rcp.PoolSize,
		MinIdleConns:       rcp.MinIdleConns,
		MaxConnAge:         HumanDuration(rcp.MaxConnAge),
		PoolTimeout:        HumanDuration(rcp.PoolTimeout),
		IdleTimeout:        HumanDuration(rcp.IdleTimeout),
		IdleCheckFrequency: HumanDuration(rcp.IdleCheckFrequency),
	}, nil
}

func (rcp *RedisClusterPool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type alias struct {
		PoolSize           int    `yaml:"poolSize" json:"pool_size" validate:"numeric"`
		MinIdleConns       int    `yaml:"minIdleConns" json:"min_idle_conns" validate:"numeric"`
		MaxConnAge         string `yaml:"maxConnAge" json:"max_conn_age"`
		PoolTimeout        string `yaml:"poolTimeout" json:"pool_timeout"`
		IdleTimeout        string `yaml:"idleTimeout" json:"idle_timeout"`
		IdleCheckFrequency string `yaml:"idleCheckTimeout" json:"idle_check_timeout"`
	}

	var tmp alias
	err := unmarshal(&tmp)
	if err != nil {
		return err
	}

	if rcp == nil {
		*rcp = RedisClusterPool{}
	}

	rcp.PoolSize = tmp.PoolSize
	rcp.MinIdleConns = tmp.MinIdleConns

	rcp.MaxConnAge, err = str2duration.ParseDuration(tmp.MaxConnAge)
	if err != nil {
		return err
	}

	rcp.PoolTimeout, err = str2duration.ParseDuration(tmp.PoolTimeout)
	if err != nil {
		return err
	}

	rcp.IdleTimeout, err = str2duration.ParseDuration(tmp.IdleTimeout)
	if err != nil {
		return err
	}

	rcp.IdleCheckFrequency, err = str2duration.ParseDuration(tmp.IdleCheckFrequency)
	if err != nil {
		return err
	}

	return nil
}

func (rcp *RedisClusterPool) MarshalJSON() ([]byte, error) {
	type alias struct {
		PoolSize           int    `yaml:"poolSize" json:"pool_size" validate:"numeric"`
		MinIdleConns       int    `yaml:"minIdleConns" json:"min_idle_conns" validate:"numeric"`
		MaxConnAge         string `yaml:"maxConnAge" json:"max_conn_age"`
		PoolTimeout        string `yaml:"poolTimeout" json:"pool_timeout"`
		IdleTimeout        string `yaml:"idleTimeout" json:"idle_timeout"`
		IdleCheckFrequency string `yaml:"idleCheckTimeout" json:"idle_check_timeout"`
	}

	if rcp == nil {
		*rcp = RedisClusterPool{}
	}

	return json.Marshal(alias{
		PoolSize:           rcp.PoolSize,
		MinIdleConns:       rcp.MinIdleConns,
		MaxConnAge:         HumanDuration(rcp.MaxConnAge),
		PoolTimeout:        HumanDuration(rcp.PoolTimeout),
		IdleTimeout:        HumanDuration(rcp.IdleTimeout),
		IdleCheckFrequency: HumanDuration(rcp.IdleCheckFrequency),
	})
}

func (rcp *RedisClusterPool) UnmarshalJSON(data []byte) (err error) {
	type alias struct {
		PoolSize           int    `yaml:"poolSize" json:"pool_size" validate:"numeric"`
		MinIdleConns       int    `yaml:"minIdleConns" json:"min_idle_conns" validate:"numeric"`
		MaxConnAge         string `yaml:"maxConnAge" json:"max_conn_age"`
		PoolTimeout        string `yaml:"poolTimeout" json:"pool_timeout"`
		IdleTimeout        string `yaml:"idleTimeout" json:"idle_timeout"`
		IdleCheckFrequency string `yaml:"idleCheckTimeout" json:"idle_check_timeout"`
	}

	var tmp alias
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if rcp == nil {
		*rcp = RedisClusterPool{}
	}

	rcp.PoolSize = tmp.PoolSize
	rcp.MinIdleConns = tmp.MinIdleConns

	rcp.MaxConnAge, err = str2duration.ParseDuration(tmp.MaxConnAge)
	if err != nil {
		return err
	}

	rcp.PoolTimeout, err = str2duration.ParseDuration(tmp.PoolTimeout)
	if err != nil {
		return err
	}

	rcp.IdleTimeout, err = str2duration.ParseDuration(tmp.IdleTimeout)
	if err != nil {
		return err
	}

	rcp.IdleCheckFrequency, err = str2duration.ParseDuration(tmp.IdleCheckFrequency)
	if err != nil {
		return err
	}

	return nil
}
