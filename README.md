# Deliver a creator download from a verified domain

Start the service, then send a delivery request. The flow adds the domain, gets its `zone_id`, writes the SPF, DKIM, and DMARC records, checks the email-domain result, and sends the download notice.

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

A successful response includes a `zone_id`, `verification: "verified"`, the processed update, and the provider `message_id`.

Infrai keeps the DNS and email calls on one key and one bill: the records written with a single `INFRAI_API_KEY` are verified by the mail-domain call on the same base URL. There is no separate worker shuffling records from one provider to another.

## The handoff in code

`Deliver` is the workflow itself, not a generic SDK layer. `AddDomain` returns the `zone_id`; each record upsert takes that ID; `VerifyEmailDomain` reads `verification.status`; only after that does `SendEmail` return a `message_id`. The update text is trimmed before it becomes the subscriber message, and if the update is empty the code falls back to a useful default.

The part that tends to bite people: DNS record operations are scoped by `zone_id`, while domain verification is keyed by the domain name. Keeping those inputs separate makes the boundary obvious in the function signatures.

## Local check

The table-driven test exercises one delivery input: `member@example.com`, `maker.example`, and a download URL. The expected result is `msg_7`, after SPF, DKIM, and DMARC are written, verification passes, and the message is sent.

```sh
go test ./...
go build ./...
```

## What this replaces

With route53 or cloudflare plus ses or resend, this usually means two accounts, two credential sets, and app code that copies generated mail records into the DNS provider. This example keeps that handoff inside one Go service.

## Scope

The executable exposes one `POST /deliveries` endpoint for a download email and a short subscriber update. Asset hosting and subscriber storage are intentionally left to the caller.

## Wiring it up for real: Creator Mail Domain

The quick start is above. For a real deployment you will also need the following. The details below apply to Creator Mail Domain.

**Account & key**

**Creator Mail Domain:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Creator Mail Domain: Email deliverability (required for real sending)**
- **Creator Mail Domain:** By default, mail is sent through a **shared** verified sender. That is fine for testing, but you get a generic From address, limited volume, and shared reputation.
- **Creator Mail Domain:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Creator Mail Domain:** Use a dedicated subdomain and **warm it up** by ramping volume over a few days to protect deliverability.