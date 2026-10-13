# nx

nx generates BIND configuration and zone files, IP lists, and a DNS search index
from NetBox prefix and IP address data.

This is a custom tool tailored to our club network with distributed name servers.
Each home network's BIND server is authoritative for its own zones, keeping at
least those zones available locally when other parts of the network are unavailable.

Zone distribution, deployment, and server operation are handled by our own
separate tool, which is not public. nx is not an “install and you have DNS from
NetBox” solution, and no support is provided. Using it elsewhere requires time to
understand the code and build the surrounding tooling.

## Input and configuration

NetBox custom fields control which prefixes and addresses participate and how
they are mapped. See the [custom-field documentation](docs/custom-fields.md) for
field definitions, validation, and inheritance rules. Address settings inherit
from the prefix used to fetch them and can override it; a feature must first be
enabled on that prefix.

The tool reads `config.json`; [config.example.json](config.example.json) shows the
NetBox connection and nameserver/zone configuration. Output is written under
`generated/` using the templates in `templates/`.

## Modules

### BIND Config & Zones

Writes zone files to `generated/zones/` and configuration for each configured
nameserver to `generated/bind-config/`, assigning primary and secondary roles
according to `config.json`.

An address's name comes from NetBox's `dns_name`, falling back to its description.
It is lowercased and cut at the first space; unnamed addresses get an
`unknown-static-N` name. A matching forward-zone suffix is removed from the name.
Forward zones with more than two labels are flattened: with
`nx_dns_forward_zone=home.example.org`, `host.home.example.org` becomes the record
`host.home` in the zone `example.org`.

| NetBox data                  | Generated DNS records and zone                                                                   |
|------------------------------|--------------------------------------------------------------------------------------------------|
| IPv4 / IPv6 address          | A / AAAA in the effective, flattened forward zone                                                |
| `nx_dns_cnames`              | CNAMEs in the same forward zone, targeting the address's name                                    |
| `nx_dns_reverse_zone` (CIDR) | PTR in the corresponding `in-addr.arpa` / `ip6.arpa` zone, targeting the normalized forward name |

`nx_dns_forward_disabled` suppresses A, AAAA, CNAME, and search-index entries while
allowing PTR records. `nx_dns_enabled=false` suppresses all DNS records for the address.

For example, an enabled address `192.0.2.10/24` named `host.home.example.org`,
with forward zone `home.example.org`, reverse zone `192.0.2.0/24`, and CNAME
`app`, produces these zone excerpts (generated comments omitted).

`generated/zones/example.org.db`:

```dns
$TTL 120
@ IN SOA ns1.example.org. admin.example.org. (
    261013001 ; serial
    900       ; refresh
    900       ; retry
    172800    ; expire
    600       ; negative cache TTL
)
@ NS ns1.example.org.
host.home A 192.0.2.10
app CNAME host.home
```

`generated/zones/2.0.192.in-addr.arpa.db` has its own SOA and NS records, followed by:

```dns
10 PTR host.home.example.org.
```

With `ns1.example.org` configured as primary for `example.org`, its generated
configuration includes the following (ACL definitions and other zones omitted):

```bind
zone "example.org" {
    type master;
    file "/etc/bind/zones/example.org.db";
    allow-transfer { "nx-secondary-acl"; };
    also-notify { "nx-secondary-primaries"; };
    notify yes;
};
```

### Search Index

Writes `generated/search-index.json` and its `.gz` companion from the same data
used to generate forward zones. The index contains A/AAAA records with hostnames,
IP addresses, CNAME aliases, source prefixes, and zones, plus zone serials,
nameservers, and SOA contacts. PTR-only entries are excluded.

For the address above, `generated/search-index.json` looks like this; the `.gz`
file contains the same JSON compressed with gzip:

```json
{
  "version": 1,
  "generated_at": "2026-10-13T00:02:00Z",
  "zones": [
    {
      "name": "example.org",
      "serial": 261013001,
      "nameserver": "ns1.example.org",
      "contact": "admin@example.org"
    }
  ],
  "records": [
    {
      "hostname": "host.home.example.org",
      "type": "A",
      "value": "192.0.2.10",
      "aliases": [
        "app.example.org"
      ],
      "prefix": "192.0.2.0/24",
      "zone": "example.org"
    }
  ]
}
```

### IP Lists

Groups enabled addresses by `nx_ip_lists`, writing one
`generated/ipl/<list>.ipl.txt` per list. Entries are individual IP addresses without
CIDR masks; an address can belong to multiple lists.

With IP lists enabled and `nx_ip_lists=["home"]`, the same address produces
`generated/ipl/home.ipl.txt` (header/footer comments omitted):

```text
192.0.2.10
```