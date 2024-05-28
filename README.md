### network-event-broker
----
A daemon configures network and executes scripts on network events such as `systemd-networkd's` [DBus](https://www.freedesktop.org/wiki/Software/dbus/) events,
`dhclient` gains lease lease. It also watches when

1. An address getting added/removed/modified.
2. Links added/removed.

```network-event-broker``` creates 

- link state directories ```carrier.d```,  ```configured.d```,  ```degraded.d```  ```no-carrier.d```  ```routable.d``` 
-  manager state dir ```manager.d``` 
-  `routes.d` (when routes gets modfied)

```bash
╭─root@Zeus1 /etc  
╰─➤  tree network-broker 
network-broker
├── carrier.d
├── configured.d
├── degraded.d
├── manager.d
├── network-broker.toml
├── no-carrier.d
```
 
in ```/etc/network-broker```. Executable scripts can be placed into directories.

Use cases:

How to run a command when get a new address is acquired via DHCP ?

1. `systemd-networkd's`
 Scripts are executed when the daemon receives the relevant event from `systemd-networkd`. See [networkctl](https://www.freedesktop.org/software/systemd/man/networkctl.html).


```bash
May 14 17:08:13 Zeus cat[273185]: OperationalState="routable"
May 14 17:08:13 Zeus cat[273185]: LINK=ens33
```

2. `dhclient`
  For `dhclient` scripts will be executed (in the dir ```routable.d```) when the `/var/lib/dhclient/dhclient.leases` file gets modified by `dhclient` and lease information is passed to the scripts as environmental arguments.

Environment variables `LINK`, `LINKINDEX=` and DHCP lease information `DHCP_LEASE=`  passed to the scripts.

#### How can I make my secondary network interface work ?

 When both interfaces are in same subnet and we have only one routing table with one GW, ie. traffic that reach via eth1 tries to leave via eth0(primary interface) which it can't. So we need to add a secondary routing table and routing policy so that the secondary interface uses the new custom routing table. Incase of static address the address and the routes already know. Incase of DHCP it's not predictable.  When `RoutingPolicyRules=` is set, `network-event-broker` automatically configures the routing policy rules `From` and `To` ensuring traffic reaches via eth1 leaves via eth1. 

#### Building from source
----

```bash

❯ make build
❯ sudo make install

```

Due to security `network-broker` runs in non root user `network-broker`. It drops all privileges except CAP_NET_ADMIN and CAP_SYS_ADMIN.

```bash
❯  useradd -M -s /usr/bin/nologin network-broker
```

### Configuration
----

Configuration file `network-broker.toml` located in ```/etc/network-broker/``` directory to manage the configuration.

The `[System]` section takes following Keys:
``` bash

LogLevel=
```
Specifies the log level. Takes one of `info`, `warn`, `error`, `debug` and `fatal`. Defaults to `info`.

```bash

Generator= 
```
Specifies the network event generator source to listen. Takes one of `systemd-networkd` or `dhclient`. Defaults to `systemd-networkd`.


The `[Network]` section takes following Keys:

```bash

Links=
```
A whitespace-separated list of links whose events should be monitored. Defaults to unset.

```bash

RoutingPolicyRules=
```
A whitespace-separated list of links for which routing policy rules would be configured per address. When set, `network-broker` automatically adds routing policy rules `from` and `to` in another routing table `(ROUTE_TABLE_BASE = 9999 + ifindex)`. When these addresses are removed, the routing policy rules are also dropped. Defaults to unset.

```bash
EmitJSON=
```
A boolean. When true, JSON format data will be emitted via envorment variable `JSON=` Applies only for `systemd-networkd`. Defaults to true.

```json
{
  "Index": 3,
  "MTU": 1500,
  "TxQLen": 1000,
  "Name": "ens37",
  "AlternativeNames": "",
  "HardwareAddr": "00:0c:29:5f:d1:43",
  "Flags": "up|broadcast|multicast",
  "RawFlags": 69699,
  "ParentIndex": 0,
  "MasterIndex": 0,
  "Namespace": "",
  "Alias": "",
  "Statistics": {
    "RxPackets": 573564,
    "TxPackets": 373642,
    "RxBytes": 540984229,
    "TxBytes": 65923722,
    "RxErrors": 0,
    "TxErrors": 0,
    "RxDropped": 0,
    "TxDropped": 0,
    "Multicast": 0,
    "Collisions": 0,
    "RxLengthErrors": 0,
    "RxOverErrors": 0,
    "RxCrcErrors": 0,
    "RxFrameErrors": 0,
    "RxFifoErrors": 0,
    "RxMissedErrors": 0,
    "TxAbortedErrors": 0,
    "TxCarrierErrors": 0,
    "TxFifoErrors": 0,
    "TxHeartbeatErrors": 0,
    "TxWindowErrors": 0,
    "RxCompressed": 0,
    "TxCompressed": 0
  },
  "Promisc": 0,
  "Xdp": {
    "Fd": 0,
    "Attached": false,
    "Flags": 0,
    "ProgId": 0
  },
  "EncapType": "ether",
  "Protinfo": "",
  "OperState": "up",
  "NetNsID": 0,
  "NumTxQueues": 1,
  "NumRxQueues": 1,
  "GSOMaxSize": 65536,
  "GSOMaxSegs": 65535,
  "Group": 0,
  "Slave": "",
  "KernelOperState": "up",
  "AddressState": "routable",
  "CarrierState": "carrier",
  "Driver": "e1000",
  "IPv4AddressState": "routable",
  "IPv6AddressState": "off",
  "LinkFile": "",
  "Model": "82545EM Gigabit Ethernet Controller (Copper)",
  "OnlineState": "online",
  "OperationalState": "routable",
  "Path": "pci-0000:02:05.0",
  "SetupState": "configuring",
  "Type": "ether",
  "Vendor": "Intel Corporation",
  "ProductID": "100f",
  "Manufacturer": "",
  "NetworkFile": "/etc/systemd/network/10-ens37.network",
  "DNS": [
    "172.16.130.2"
  ],
  "Domains": null,
  "NTP": null,
  "Address": [
    {
      "IP": "172.16.130.144",
      "Mask": 24,
      "Label": "ens37",
      "Flags": 0,
      "Scope": 0,
      "Peer": "",
      "Broadcast": "172.16.130.255",
      "PreferedLft": 1800,
      "ValidLft": 1800
    },
    {
      "IP": "fe80::20c:29ff:fe5f:d143",
      "Mask": 64,
      "Label": "",
      "Flags": 192,
      "Scope": 253,
      "Peer": "",
      "Broadcast": "",
      "PreferedLft": 4294967295,
      "ValidLft": 4294967295
    }
  ],
  "Routes": [
    {
      "Scope": 0,
      "Dst": {
        "IP": "",
        "Mask": 0
      },
      "Src": "172.16.130.144",
      "Gw": "172.16.130.2",
      "MultiPath": "",
      "Protocol": 16,
      "Priority": 1024,
      "Table": 254,
      "Type": 1,
      "Tos": 0,
      "Flags": null,
      "MPLSDst": "",
      "NewDst": "",
      "Encap": "",
      "MTU": 0,
      "AdvMSS": 0,
      "Hoplimit": 0
    },
    {
      "Scope": 253,
      "Dst": {
        "IP": "172.16.130.0",
        "Mask": 24
      },
      "Src": "172.16.130.144",
      "Gw": "",
      "MultiPath": "",
      "Protocol": 2,
      "Priority": 1024,
      "Table": 254,
      "Type": 1,
      "Tos": 0,
      "Flags": null,
      "MPLSDst": "",
      "NewDst": "",
      "Encap": "",
      "MTU": 0,
      "AdvMSS": 0,
      "Hoplimit": 0
    },
    {
      "Scope": 253,
      "Dst": {
        "IP": "172.16.130.2",
        "Mask": 32
      },
      "Src": "172.16.130.144",
      "Gw": "",
      "MultiPath": "",
      "Protocol": 16,
      "Priority": 1024,
      "Table": 254,
      "Type": 1,
      "Tos": 0,
      "Flags": null,
      "MPLSDst": "",
      "NewDst": "",
      "Encap": "",
      "MTU": 0,
      "AdvMSS": 0,
      "Hoplimit": 0
    },
    {
      "Scope": 0,
      "Dst": {
        "IP": "fe80::",
        "Mask": 64
      },
      "Src": "",
      "Gw": "",
      "MultiPath": "",
      "Protocol": 2,
      "Priority": 256,
      "Table": 254,
      "Type": 1,
      "Tos": 0,
      "Flags": null,
      "MPLSDst": "",
      "NewDst": "",
      "Encap": "",
      "MTU": 0,
      "AdvMSS": 0,
      "Hoplimit": 0
    }
  ]
}

```

```bash
UseDNS=
```
A boolean. When true, the DNS server will be se to `systemd-resolved` vis DBus. Applies only for DHClient. Defaults to false.

```bash
UseDomain=
```
A boolean. When true, the DNS domains will be sent to `systemd-resolved` vis DBus. Applies only for DHClient. Defaults to false.

```bash
UseHostname=
```
A boolean. When true, the host name be sent to `systemd-hostnamed` vis DBus. Applies only for DHClient. Defaults to false.

```bash
❯ sudo cat /etc/network-broker/network-broker.toml 
[System]
LogLevel="debug"
Generator="systemd-networkd"

[Network]
Links="eth0 eth1"
RoutingPolicyRules="eth1"
UseDNS="true"
UseDomain="true"
EmitJSON="true"

```

```bash

❯ systemctl status network-broker.service
● network-broker.service - A daemon configures network upon events
     Loaded: loaded (/usr/lib/systemd/system/network-broker.service; disabled; vendor preset: disabled)
     Active: active (running) since Thu 2022-06-03 22:22:38 CEST; 3h 13min ago
       Docs: man:networkd-broker.conf(5)
   Main PID: 572392 (network-broker)
      Tasks: 7 (limit: 9287)
     Memory: 6.2M
        CPU: 319ms
     CGroup: /system.slice/network-broker.service
             └─572392 /usr/bin/network-broker

Jun 04 01:36:04 Zeus network-broker[572392]: [info] 2022/06/04 01:36:04 Link='ens33' ifindex='2' changed state 'OperationalState'="carrier"
Jun 04 01:36:04 Zeus network-broker[572392]: [info] 2022/06/04 01:36:04 Link='' ifindex='1' changed state 'OperationalState'="carrier"

```
DBus signals generated by ```systemd-networkd```
```bash

&{:1.683 /org/freedesktop/network1/link/_32 org.freedesktop.DBus.Properties.PropertiesChanged [org.freedesktop.network1.Link map[AdministrativeState:"configured"] []] 10}
```

```
‣ Type=signal  Endian=l  Flags=1  Version=1 Cookie=24  Timestamp="Sun 2022-05-16 08:06:05.905781 UTC"
  Sender=:1.292  Path=/org/freedesktop/network1  Interface=org.freedesktop.DBus.Properties  Member=PropertiesChanged
  UniqueName=:1.292
  MESSAGE "sa{sv}as" {
          STRING "org.freedesktop.network1.Manager";
          ARRAY "{sv}" {
                  DICT_ENTRY "sv" {
                          STRING "OperationalState";
                          VARIANT "s" {
                                  STRING "degraded";
                          };
                  };
          };
          ARRAY "s" {
          };
  };

```


#### Contributing
----

The **Network Event Broker** project team welcomes contributions from the community. If you wish to contribute code and you have not signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq).

slack channel [#photon](https://code.vmware.com/web/code/join).

#### License
----

[Apache-2.0](https://spdx.org/licenses/Apache-2.0.html)
