# Release Verification

Nightward releases are human-gated and signed.

## Verify signed checksums

```sh
cosign verify-blob \
  --certificate-identity-regexp 'https://github.com/JSONbored/nightward/.github/workflows/release.yml@refs/tags/v.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt
```

## Verify archive checksum

```sh
sha256sum -c checksums.txt --ignore-missing
```

## NPM launcher

The npm package downloads the matching GitHub Release archive on first run and verifies it against `checksums.txt`.

After install:

```sh
npx @jsonbored/nightward --version
nw doctor --json
```

> [!IMPORTANT]
> Prefer GitHub Release artifacts plus signed checksum verification when supply-chain assurance matters more than install convenience.
