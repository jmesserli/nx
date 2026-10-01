# NetBox custom-field migration

NX reads the existing REST endpoints. Prefixes and IP addresses may use these optional custom fields alongside existing
`nx:*` tags:

| Name                      | NetBox type           | Legacy tag                          |
|---------------------------|-----------------------|-------------------------------------|
| `nx_dns_enabled`          | Boolean               | `nx:dns:enable[true]`               |
| `nx_dns_forward_disabled` | Boolean               | None (new control)                  |
| `nx_dns_forward_zone`     | Text                  | `nx:dns:forward_zone[example.com]`  |
| `nx_dns_reverse_zone`     | Text (CIDR)           | `nx:dns:reverse_zone[192.0.2.0/24]` |
| `nx_dns_cnames`           | JSON array of strings | Repeated `nx:dns:cname[alias]`      |
| `nx_ip_lists_enabled`     | Boolean               | `nx:ipl:enable[true]`               |
| `nx_ip_lists`             | JSON array of strings | Repeated `nx:ipl:list[internal]`    |

Assign the fields to both `ipam.prefix` and `ipam.ipaddress`. Leave them optional and without defaults during migration.
Confirm that your deployed NetBox version returns unset fields as null or missing, booleans as JSON booleans, and the
JSON list fields as arrays of strings. Selection fields are not used by this contract.

For each property, precedence is:

1. IP-address custom field
2. IP-address legacy tag
3. Prefix custom field
4. Prefix legacy tag
5. False, empty string, or no list

Missing/null means inherit or use the legacy tag. Explicit `false` overrides an inherited boolean;
`""` clears a string if returned by the API (see the forward-zone exception below); `[]` clears a list. Lists replace
inherited lists instead of merging. Unknown custom fields are ignored. Incorrect JSON types fail decoding before
generation rather than silently falling back to tags.

Prefix enablement remains the fetch/routing gate: an IP address cannot activate DNS or IP lists on a disabled prefix.
Addresses inherit from the prefix used to fetch them; this does not introduce inheritance between nested prefixes.

Forward generation is enabled by default for DNS-enabled addresses with a forward zone. Set `nx_dns_forward_disabled` to
`true` on a prefix to suppress A, AAAA, CNAME, and search-index records for its addresses (for example, a DHCP pool). An
address inherits this setting when unset/null; explicit `true` suppresses an individual address, and explicit `false`
restores forward generation inside a disabled prefix. Keep this field without a NetBox default so new addresses inherit
their prefix. The forward zone remains available for PTR targets and hostname normalization.
`nx_dns_enabled=false` still disables all DNS generation; the forward flag cannot override that gate. The reverse zone
continues to select which PTR records to generate.

NetBox's UI converts blank text custom fields to null, which means inheritance in NX. Use the Boolean flag for PTR-only
addresses instead of clearing the forward zone. The resolver translates legacy
`nx:dns:forward_zone[]` into `DNSForwardDisabled=true`, retaining the inherited forward zone for PTR targets. An
explicit empty custom zone is normalized the same way. A nonempty custom zone on the same object replaces the legacy
clearing effect, and an explicit custom disable flag takes precedence. A child zone can undo clearing inherited from a
prefix's legacy tag, but continues to inherit an explicit prefix disable flag. If no forward zone is available, PTR
targets use the address name alone. The generator uses only the resolved zone and disable flag; it has no legacy
fallback.

Populate custom fields from existing tags and compare generated output in an isolated directory before switching
production. Keep tag fallback until all objects have migrated. No NetBox data is written by NX. After migration, remove
the legacy adapter and `tagparser` package in a separate change.
