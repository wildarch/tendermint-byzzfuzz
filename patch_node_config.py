#!/usr/bin/env python3
import toml
import json

TESTING_SERVER_ADDR = "127.0.0.1:7074"

N = 4

NODES = [f'node{i}' for i in range(N)]

NODE_ORIG_IP = {NODES[i]: f"192.167.10.{i + 2}" for i in range(N)}
ORIG_IP_NODE = {f"192.167.10.{i + 2}": NODES[i] for i in range(N)}
NODE_ORIG_RPC_PORT = 26657
NODE_ORIG_P2P_PORT = 26656
NODE_RPC_PORT = {}
NODE_P2P_PORT = {}
CONTROLLER_LADDR = {}
for i in range(N):
    NODE_RPC_PORT[NODES[i]] = 26650 + i
    NODE_P2P_PORT[NODES[i]] = 26660 + i
    CONTROLLER_LADDR[NODES[i]] = 8880 + i

# Update addrbook
for node in NODES:
    addrbook_file = open(f"node_homes/{node}/config/addrbook.json", "r")
    addrbook = json.load(addrbook_file)
    addrbook_file.close()

    for addr in addrbook["addrs"]:
        # Set src port
        addr["src"]["port"] = NODE_P2P_PORT[node]
        ip = addr["addr"]["ip"]
        if ip not in ORIG_IP_NODE:
            continue
        # Set target ip:port
        target_node = ORIG_IP_NODE[ip]
        addr["addr"]["ip"] = "127.0.0.1"
        addr["addr"]["port"] = NODE_P2P_PORT[target_node]
    
    addrbook_file = open(f"node_homes/{node}/config/addrbook.json", "w")
    json.dump(addrbook, addrbook_file, indent="\t")
    addrbook_file.close()

# Update config.toml
for node in NODES:
    config = toml.load(f"node_homes/{node}/config/config.toml")

    config["rpc"]["laddr"] = f"tcp://127.0.0.1:{NODE_RPC_PORT[node]}"
    config["p2p"]["controller-master-addr"] = TESTING_SERVER_ADDR
    config["p2p"]["controller-listen-addr"] = f"127.0.0.1:{CONTROLLER_LADDR[node]}"
    config["p2p"]["laddr"] = f"tcp://127.0.0.1:{NODE_P2P_PORT[node]}"
    peers = config["p2p"]["persistent-peers"]
    for peer in NODES:
        peers = peers.replace(f"{NODE_ORIG_IP[peer]}:{NODE_ORIG_P2P_PORT}", f"127.0.0.1:{NODE_P2P_PORT[peer]}")
    config["p2p"]["persistent-peers"] = peers

    config_file = open(f"node_homes/{node}/config/config.toml", "w")
    toml.dump(config, config_file)
    config_file.close()