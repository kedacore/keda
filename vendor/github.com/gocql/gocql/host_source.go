package gocql

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type nodeState int32

func (n nodeState) String() string {
	if n == NodeUp {
		return "UP"
	} else if n == NodeDown {
		return "DOWN"
	}
	return fmt.Sprintf("UNKNOWN_%d", n)
}

const (
	NodeUp nodeState = iota
	NodeDown
)

type cassVersion struct {
	Major, Minor, Patch int
}

func (c *cassVersion) Set(v string) error {
	if v == "" {
		return nil
	}

	return c.UnmarshalCQL(nil, []byte(v))
}

func (c *cassVersion) UnmarshalCQL(info TypeInfo, data []byte) error {
	return c.unmarshal(data)
}

func (c *cassVersion) unmarshal(data []byte) error {
	version := strings.TrimSuffix(string(data), "-SNAPSHOT")
	version = strings.TrimPrefix(version, "v")
	v := strings.Split(version, ".")

	if len(v) < 2 {
		return fmt.Errorf("invalid version string: %s", data)
	}

	var err error
	c.Major, err = strconv.Atoi(v[0])
	if err != nil {
		return fmt.Errorf("invalid major version %v: %v", v[0], err)
	}

	c.Minor, err = strconv.Atoi(v[1])
	if err != nil {
		return fmt.Errorf("invalid minor version %v: %v", v[1], err)
	}

	if len(v) > 2 {
		c.Patch, err = strconv.Atoi(v[2])
		if err != nil {
			return fmt.Errorf("invalid patch version %v: %v", v[2], err)
		}
	}

	return nil
}

func (c cassVersion) Before(major, minor, patch int) bool {
	// We're comparing us (cassVersion) with the provided version (major, minor, patch)
	// We return true if our version is lower (comes before) than the provided one.
	if c.Major < major {
		return true
	} else if c.Major == major {
		if c.Minor < minor {
			return true
		} else if c.Minor == minor && c.Patch < patch {
			return true
		}

	}
	return false
}

func (c cassVersion) AtLeast(major, minor, patch int) bool {
	return !c.Before(major, minor, patch)
}

func (c cassVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", c.Major, c.Minor, c.Patch)
}

func (c cassVersion) nodeUpDelay() time.Duration {
	if c.Major >= 2 && c.Minor >= 2 {
		// CASSANDRA-8236
		return 0
	}

	return 10 * time.Second
}

type HostInfo struct {
	// TODO(zariel): reduce locking maybe, not all values will change, but to ensure
	// that we are thread safe use a mutex to access all fields.
	mu               sync.RWMutex
	hostname         string
	peer             net.IP
	broadcastAddress net.IP
	listenAddress    net.IP
	rpcAddress       net.IP
	preferredIP      net.IP
	connectAddress   net.IP
	port             int
	dataCenter       string
	rack             string
	hostId           string
	workload         string
	graph            bool
	dseVersion       string
	partitioner      string
	clusterName      string
	version          cassVersion
	state            nodeState
	schemaVersion    string
	tokens           []string
}

func (h *HostInfo) Equal(host *HostInfo) bool {
	if h == host {
		// prevent rlock reentry
		return true
	}

	return h.ConnectAddress().Equal(host.ConnectAddress())
}

func (h *HostInfo) Peer() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.peer
}

func (h *HostInfo) invalidConnectAddr() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	addr, _ := h.connectAddressLocked()
	return !validIpAddr(addr)
}

func validIpAddr(addr net.IP) bool {
	return addr != nil && !addr.IsUnspecified()
}

func (h *HostInfo) connectAddressLocked() (net.IP, string) {
	if validIpAddr(h.connectAddress) {
		return h.connectAddress, "connect_address"
	} else if validIpAddr(h.rpcAddress) {
		return h.rpcAddress, "rpc_adress"
	} else if validIpAddr(h.preferredIP) {
		// where does perferred_ip get set?
		return h.preferredIP, "preferred_ip"
	} else if validIpAddr(h.broadcastAddress) {
		return h.broadcastAddress, "broadcast_address"
	} else if validIpAddr(h.peer) {
		return h.peer, "peer"
	}
	return net.IPv4zero, "invalid"
}

// nodeToNodeAddress returns address broadcasted between node to nodes.
// It's either `broadcast_address` if host info is read from system.local or `peer` if read from system.peers.
// This IP address is also part of CQL Event emitted on topology/status changes,
// but does not uniquely identify the node in case multiple nodes use the same IP address.
func (h *HostInfo) nodeToNodeAddress() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if validIpAddr(h.broadcastAddress) {
		return h.broadcastAddress
	} else if validIpAddr(h.peer) {
		return h.peer
	}
	return net.IPv4zero
}

// Returns the address that should be used to connect to the host.
// If you wish to override this, use an AddressTranslator or
// use a HostFilter to SetConnectAddress()
func (h *HostInfo) ConnectAddress() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if addr, _ := h.connectAddressLocked(); validIpAddr(addr) {
		return addr
	}
	panic(fmt.Sprintf("no valid connect address for host: %v. Is your cluster configured correctly?", h))
}

func (h *HostInfo) SetConnectAddress(address net.IP) *HostInfo {
	// TODO(zariel): should this not be exported?
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connectAddress = address
	return h
}

func (h *HostInfo) BroadcastAddress() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.broadcastAddress
}

func (h *HostInfo) ListenAddress() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.listenAddress
}

func (h *HostInfo) RPCAddress() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rpcAddress
}

func (h *HostInfo) PreferredIP() net.IP {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.preferredIP
}

func (h *HostInfo) DataCenter() string {
	h.mu.RLock()
	dc := h.dataCenter
	h.mu.RUnlock()
	return dc
}

func (h *HostInfo) Rack() string {
	h.mu.RLock()
	rack := h.rack
	h.mu.RUnlock()
	return rack
}

func (h *HostInfo) HostID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hostId
}

func (h *HostInfo) SetHostID(hostID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hostId = hostID
}

func (h *HostInfo) WorkLoad() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.workload
}

func (h *HostInfo) Graph() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.graph
}

func (h *HostInfo) DSEVersion() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.dseVersion
}

func (h *HostInfo) Partitioner() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.partitioner
}

func (h *HostInfo) ClusterName() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.clusterName
}

func (h *HostInfo) Version() cassVersion {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.version
}

func (h *HostInfo) State() nodeState {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.state
}

func (h *HostInfo) setState(state nodeState) *HostInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.state = state
	return h
}

func (h *HostInfo) Tokens() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tokens
}

func (h *HostInfo) Port() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.port
}

func (h *HostInfo) update(from *HostInfo) {
	if h == from {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	from.mu.RLock()
	defer from.mu.RUnlock()

	// autogenerated do not update
	if h.peer == nil {
		h.peer = from.peer
	}
	if h.broadcastAddress == nil {
		h.broadcastAddress = from.broadcastAddress
	}
	if h.listenAddress == nil {
		h.listenAddress = from.listenAddress
	}
	if h.rpcAddress == nil {
		h.rpcAddress = from.rpcAddress
	}
	if h.preferredIP == nil {
		h.preferredIP = from.preferredIP
	}
	if h.connectAddress == nil {
		h.connectAddress = from.connectAddress
	}
	if h.port == 0 {
		h.port = from.port
	}
	if h.dataCenter == "" {
		h.dataCenter = from.dataCenter
	}
	if h.rack == "" {
		h.rack = from.rack
	}
	if h.hostId == "" {
		h.hostId = from.hostId
	}
	if h.workload == "" {
		h.workload = from.workload
	}
	if h.dseVersion == "" {
		h.dseVersion = from.dseVersion
	}
	if h.partitioner == "" {
		h.partitioner = from.partitioner
	}
	if h.clusterName == "" {
		h.clusterName = from.clusterName
	}
	if h.version == (cassVersion{}) {
		h.version = from.version
	}
	if h.tokens == nil {
		h.tokens = from.tokens
	}
}

func (h *HostInfo) IsUp() bool {
	return h != nil && h.State() == NodeUp
}

func (h *HostInfo) HostnameAndPort() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hostname == "" {
		addr, _ := h.connectAddressLocked()
		h.hostname = addr.String()
	}
	return net.JoinHostPort(h.hostname, strconv.Itoa(h.port))
}

func (h *HostInfo) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	connectAddr, source := h.connectAddressLocked()
	return fmt.Sprintf("[HostInfo hostname=%q connectAddress=%q peer=%q rpc_address=%q broadcast_address=%q "+
		"preferred_ip=%q connect_addr=%q connect_addr_source=%q "+
		"port=%d data_centre=%q rack=%q host_id=%q version=%q state=%s num_tokens=%d]",
		h.hostname, h.connectAddress, h.peer, h.rpcAddress, h.broadcastAddress, h.preferredIP,
		connectAddr, source,
		h.port, h.dataCenter, h.rack, h.hostId, h.version, h.state, len(h.tokens))
}

// Polls system.peers at a specific interval to find new hosts
type ringDescriber struct {
	session         *Session
	mu              sync.Mutex
	prevHosts       []*HostInfo
	prevPartitioner string
}

// Returns true if we are using system_schema.keyspaces instead of system.schema_keyspaces
func checkSystemSchema(control *controlConn) (bool, error) {
	iter := control.query("SELECT * FROM system_schema.keyspaces")
	if err := iter.err; err != nil {
		if errf, ok := err.(*errorFrame); ok {
			if errf.code == ErrCodeSyntax {
				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}

// Given a map that represents a row from either system.local or system.peers
// return as much information as we can in *HostInfo
func (s *Session) hostInfoFromMap(row map[string]interface{}, host *HostInfo) (*HostInfo, error) {
	const assertErrorMsg = "Assertion failed for %s"
	var ok bool

	// Default to our connected port if the cluster doesn't have port information
	for key, value := range row {
		switch key {
		case "data_center":
			host.dataCenter, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "data_center")
			}
		case "rack":
			host.rack, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "rack")
			}
		case "host_id":
			hostId, ok := value.(UUID)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "host_id")
			}
			host.hostId = hostId.String()
		case "release_version":
			version, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "release_version")
			}
			host.version.Set(version)
		case "peer":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "peer")
			}
			host.peer = net.ParseIP(ip)
		case "cluster_name":
			host.clusterName, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "cluster_name")
			}
		case "partitioner":
			host.partitioner, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "partitioner")
			}
		case "broadcast_address":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "broadcast_address")
			}
			host.broadcastAddress = net.ParseIP(ip)
		case "preferred_ip":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "preferred_ip")
			}
			host.preferredIP = net.ParseIP(ip)
		case "rpc_address":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "rpc_address")
			}
			host.rpcAddress = net.ParseIP(ip)
		case "native_address":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "native_address")
			}
			host.rpcAddress = net.ParseIP(ip)
		case "listen_address":
			ip, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "listen_address")
			}
			host.listenAddress = net.ParseIP(ip)
		case "native_port":
			native_port, ok := value.(int)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "native_port")
			}
			host.port = native_port
		case "workload":
			host.workload, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "workload")
			}
		case "graph":
			host.graph, ok = value.(bool)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "graph")
			}
		case "tokens":
			host.tokens, ok = value.([]string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "tokens")
			}
		case "dse_version":
			host.dseVersion, ok = value.(string)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "dse_version")
			}
		case "schema_version":
			schemaVersion, ok := value.(UUID)
			if !ok {
				return nil, fmt.Errorf(assertErrorMsg, "schema_version")
			}
			host.schemaVersion = schemaVersion.String()
		}
		// TODO(thrawn01): Add 'port'? once CASSANDRA-7544 is complete
		// Not sure what the port field will be called until the JIRA issue is complete
	}

	ip, port := s.cfg.translateAddressPort(host.ConnectAddress(), host.port)
	host.connectAddress = ip
	host.port = port

	return host, nil
}

// Ask the control node for host info on all it's known peers
func (r *ringDescriber) getClusterPeerInfo() ([]*HostInfo, error) {
	var hosts []*HostInfo
	iter := r.session.control.withConnHost(func(ch *connHost) *Iter {
		hosts = append(hosts, ch.host)
		return ch.conn.querySystemPeers(context.TODO(), ch.host.version)
	})

	if iter == nil {
		return nil, errNoControl
	}

	rows, err := iter.SliceMap()
	if err != nil {
		// TODO(zariel): make typed error
		return nil, fmt.Errorf("unable to fetch peer host info: %s", err)
	}

	for _, row := range rows {
		// extract all available info about the peer
		host, err := r.session.hostInfoFromMap(row, &HostInfo{port: r.session.cfg.Port})
		if err != nil {
			return nil, err
		} else if !isValidPeer(host) {
			// If it's not a valid peer
			r.session.logger.Printf("Found invalid peer '%s' "+
				"Likely due to a gossip or snitch issue, this host will be ignored", host)
			continue
		}

		hosts = append(hosts, host)
	}

	return hosts, nil
}

// Return true if the host is a valid peer
func isValidPeer(host *HostInfo) bool {
	return !(len(host.RPCAddress()) == 0 ||
		host.hostId == "" ||
		host.dataCenter == "" ||
		host.rack == "" ||
		len(host.tokens) == 0)
}

// Return a list of hosts the cluster knows about
func (r *ringDescriber) GetHosts() ([]*HostInfo, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	hosts, err := r.getClusterPeerInfo()
	if err != nil {
		return r.prevHosts, r.prevPartitioner, err
	}

	var partitioner string
	if len(hosts) > 0 {
		partitioner = hosts[0].Partitioner()
	}

	return hosts, partitioner, nil
}

// Given an ip/port return HostInfo for the specified ip/port
func (r *ringDescriber) getHostInfo(hostID UUID) (*HostInfo, error) {
	var host *HostInfo
	for _, table := range []string{"system.peers", "system.local"} {
		iter := r.session.control.withConnHost(func(ch *connHost) *Iter {
			if ch.host.HostID() == hostID.String() {
				host = ch.host
				return nil
			}

			if table == "system.peers" {
				return ch.conn.querySystemPeers(context.TODO(), ch.host.version)
			} else {
				return ch.conn.query(context.TODO(), fmt.Sprintf("SELECT * FROM %s", table))
			}
		})

		if iter != nil {
			rows, err := iter.SliceMap()
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				h, err := r.session.hostInfoFromMap(row, &HostInfo{port: r.session.cfg.Port})
				if err != nil {
					return nil, err
				}

				if h.HostID() == hostID.String() {
					host = h
					break
				}
			}
		}
	}

	if host == nil {
		return nil, errors.New("unable to fetch host info: invalid control connection")
	}

	return host, nil
}

func (r *ringDescriber) refreshRing() error {
	// if we have 0 hosts this will return the previous list of hosts to
	// attempt to reconnect to the cluster otherwise we would never find
	// downed hosts again, could possibly have an optimisation to only
	// try to add new hosts if GetHosts didnt error and the hosts didnt change.
	hosts, partitioner, err := r.GetHosts()
	if err != nil {
		return err
	}

	prevHosts := r.session.ring.currentHosts()

	// TODO: move this to session
	for _, h := range hosts {
		if r.session.cfg.filterHost(h) {
			continue
		}

		if host, ok := r.session.ring.addHostIfMissing(h); !ok {
			r.session.startPoolFill(h)
		} else {
			host.update(h)
		}
		delete(prevHosts, h.HostID())
	}

	// TODO(zariel): it may be worth having a mutex covering the overall ring state
	// in a session so that everything sees a consistent state. Becuase as is today
	// events can come in and due to ordering an UP host could be removed from the cluster
	for _, host := range prevHosts {
		r.session.removeHost(host)
	}

	r.session.metadata.setPartitioner(partitioner)
	r.session.policy.SetPartitioner(partitioner)
	return nil
}
