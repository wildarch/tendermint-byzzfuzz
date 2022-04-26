Need to change IPs in range 192.167.10.*

Original node IPs were:
- 192.167.10.2
- 192.167.10.3
- 192.167.10.4
- 192.167.10.5

All used port 26656.

Mapping is on addrbook.json and config.toml.

Set `p2p.controller-master-addr` to testing server addr.

Set `rpc.laddr` to localhost and a port for that node.

Set `p2p.laddr` to localhost and a port for that node.

Update `p2p.persistent-peers`.

`p2p.controller-listen-addr` not sure?