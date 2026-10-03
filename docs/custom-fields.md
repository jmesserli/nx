# NetBox custom fields

Create these optional custom fields without defaults. Leave **Required** and **Validate uniqueness** unchecked.

| Name                      | Label                                  | Group  | Type    | Object types       |
|---------------------------|----------------------------------------|--------|---------|--------------------|
| `nx_dns_cnames`           | CNAMEs                                 | NX DNS | JSON    | IP Address         |
| `nx_dns_enabled`          | Enabled                                | NX DNS | Boolean | Prefix, IP Address |
| `nx_dns_forward_disabled` | Disable Forward DNS (A / AAAA / CNAME) | NX DNS | Boolean | Prefix, IP Address |
| `nx_dns_forward_zone`     | Forward Zone                           | NX DNS | Text    | Prefix, IP Address |
| `nx_dns_reverse_zone`     | Reverse Zone                           | NX DNS | Text    | Prefix, IP Address |
| `nx_ip_lists`             | List Names                             | NX IPL | JSON    | Prefix, IP Address |
| `nx_ip_lists_enabled`     | Enable IP List                         | NX IPL | Boolean | Prefix, IP Address |

Use arrays of strings for JSON fields, e.g. `["docker", "app1"]` for CNAMEs and `["pegnu-rackfarm"]` for list names.
Set **Validation schema** on both `nx_dns_cnames` and `nx_ip_lists` to:

```json
{
  "type": "array",
  "items": {
    "type": "string"
  }
}
```

Empty arrays remain valid so inherited lists can be cleared. Leave the fields optional so unset/null can inherit.
NetBox documents these controls
under [custom-field validation](https://netbox.readthedocs.io/en/stable/models/extras/customfield/#validation-schema).

Reverse zones are CIDRs, e.g. `192.0.2.0/24`, rather than reverse DNS domain names.
The generator keeps whole IPv4 octets (`prefix length / 8`) or IPv6 nibbles (`prefix length / 4`), rounding down.
Use aligned lengths to avoid silently generating a broader zone: IPv4 `/8`, `/16`, `/24`; IPv6 multiples of four
from `/4` through `/124`. Exclude `/0` and host-only `/32` or `/128`: the current generator needs both a zone label
and a remaining PTR label. For example, `192.0.2.0/24` becomes `2.0.192.in-addr.arpa`, and `2001:db8::/32`
becomes `8.b.d.0.1.0.0.2.ip6.arpa`.

Suggested **Validation regex** for `nx_dns_reverse_zone` (one line, accepts either family):

```regex
^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])(?:\.(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])){3}/(?:8|16|24)|(?![^/]*:::)(?![^/]*::[^/]*::)(?=(?:[^:/]*:){2})(?:[0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}/(?:[48]|[2468][048]|[13579][26]|10[048]|11[26]|12[04]))$
```

The IPv4 branch checks four octets in `0–255` and byte-aligned lengths. The IPv6 branch is deliberately approximate:
it checks hexadecimal groups, rejects triple colons and repeated `::`, and requires nibble-aligned lengths;
it does not validate every compression rule or support dotted IPv4 tails. Use network addresses with host bits zero
and a reverse zone containing the IP address; the regex does not check either relationship.

- Unset/null address fields inherit from the prefix used to fetch the address; nested prefixes do not inherit from each
  other.
  Explicit `false` overrides a prefix Boolean. Lists replace inherited lists; `[]` clears them.
- Enable DNS or IP lists on the prefix first. An address cannot activate a feature disabled on its prefix.
- For PTR-only addresses, set `nx_dns_forward_disabled=true`. This suppresses A, AAAA, CNAME, and search-index records
  while retaining the forward zone for PTR targets. An address can override a prefix disable with `false`.
  `nx_dns_enabled=false` disables all DNS generation.
- Blank text in NetBox is returned as null and therefore inherits. Use the forward-disable flag to suppress forward DNS.
  An explicit empty string returned by the API clears a zone; it does not set the forward-disable flag.
