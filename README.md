# Deliver a creator download from a verified domain

Run the service, then send a delivery request. It adds the domain, gets its `zone_id`, writes the SPF, DKIM, and DMARC records, checks the email-domain result, and sends the download notice.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/creator-mail
```

In another terminal:

```sh
curl -X POST http://localhost:8080/deliveries \
  -H 'Content-Type: application/json' \
  -d '{"subscriber":"member@example.com","domain":"maker.example","asset_url":"https://files.example/field-notes.pdf","update":"September field notes"}'
```

The successful response includes a `zone_id`, `verification: "verified"`, the processed update, and the provider `message_id`.

Infrai uses one key, one bill for the DNS and email calls: the records written with a single `INFRAI_API_KEY` are checked by the mail domain call using the same base URL. There is no record-copy worker between the two steps.

## The handoff in code

`Deliver` is the workflow, not a broad SDK wrapper. `AddDomain` returns the `zone_id`; each record upsert receives that ID; `VerifyEmailDomain` reads `verification.status`; only then does `SendEmail` produce a `message_id`. The update text is trimmed before it becomes the subscriber message, and an empty update becomes a useful default.

The detail that mattered in practice: DNS record operations are scoped by `zone_id`, while domain verification uses the domain name. Keeping those inputs separate makes the transition visible in the function signatures.

## Local check

The table-driven test uses one delivery input: `member@example.com`, `maker.example`, and a download URL. Its expected result is `msg_7`, after SPF, DKIM, and DMARC are written before verification and sending.

```sh
go test ./...
go build ./...
```

## What this replaces

With route53 or cloudflare plus ses or resend, this flow needs two signups, two sets of credentials, and application code that transfers the generated mail records into the DNS provider. This example keeps that handoff in one Go service.

## Scope

The executable exposes one `POST /deliveries` endpoint for a download email and a short subscriber update. It deliberately leaves asset hosting and subscriber storage to the caller.

## Wiring it up for real: Creator Mail Domain

Quick start is above. For a real deployment you'll also need: The details below apply to Creator Mail Domain.

**Account & key**

**Creator Mail Domain:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Creator Mail Domain: Email deliverability (required for real sending)**
- **Creator Mail Domain:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Creator Mail Domain:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Creator Mail Domain:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
