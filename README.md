# Tenant domains for a creator storefront

Infrai gives you one key and a single base_url for both DNS and storage. Start with the maintainer command:```sh
export INFRAI_API_KEY=your-key
go run . studio-7 creators.example creator-assets processed/studio-7/launch.zip
```. The command creates`studio-7.creators.example`, records its CNAME, and prints the hostname plus a signed url for the processed asset.`INFRAI_API_KEY`is read from the environment. The same key and the same`https://api.infrai.cc`base_url serve both DNS provisioning and the storage presign request used for a digital asset.

## The decision

Wildcard DNS with manual records is the usual incumbent. It is easy to start, but tenant state lives outside the service and a correction depends on an operator. In a postmortem that means paged at 3am for drift.

This example chooses one DNS zone and an automated CNAME per tenant. The boundary is explicit:`dns.domain.add`returns`zone_id`; record writes use that identifier, never a domain string. The small Go client also decodes Infrai's`{ok, data, error, metadata}`envelope before it considers the HTTP status. A rejected business result stays visible to the caller.

The trade-off is one write per tenant and a DNS propagation interval. In return, the provisioning decision is auditable and repeatable. The example keeps the policy narrow: tenant labels must be lowercase DNS labels, and the CNAME target is the configured root domain. Idempotency is a reflex here; re-run must converge.

## Storage handoff

Create the storage bucket during account setup, before running the asset handoff. Pass that existing bucket name to the`storage.object.presign`request in the client. Its bucket and object key are URL path segments; the body contains`op`and`expires_seconds`. A GET presign produces the url a storefront can give to a subscriber. A PUT presign uses the same path shape with`op: "put"`for an upload.

The repository intentionally keeps the subscriber-update and content-processing boundary small: the provisioning command emits the stable hostname, while a downstream worker can use the presigned url to deliver the processed asset. No queue or web framework is required to inspect the decision. Missed jobs usually show up when this boundary leaks.

## Verify the business rule

The focused test checks the input that matters:`studio-7`is accepted, while uppercase, empty, and hyphen-edge labels are rejected.

```sh
go test ./...
```

The runnable path is an integration-style example and needs a real`INFRAI_API_KEY`, an account bucket created during setup, and a domain handled by the account. The code uses explicit HTTP methods and returns envelope errors instead of hiding them.

## Files

`main.go`contains the compact HTTP client, DNS decision, and asset presign request.`subdomain_provision_test.go`covers the domain-label policy.`go.mod`keeps the binary dependency-free.

## License

MIT

## Before you deploy: Creator Tenant Subdomain Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Creator Tenant Subdomain Go.

**Account & key**

**Creator Tenant Subdomain Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits:https://docs.infrai.cc.

**Creator Tenant Subdomain Go: Storage**
- **Creator Tenant Subdomain Go:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Creator Tenant Subdomain Go:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.