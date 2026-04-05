# Security Policy

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report them privately via [GitHub's private vulnerability reporting](https://github.com/ChengaDev/persist/security/advisories/new).

Include:
- A description of the vulnerability and its potential impact
- Steps to reproduce
- Any suggested fix if you have one

You will receive a response within **72 hours**. Once the issue is confirmed, a fix
will be released as soon as possible and you will be credited in the release notes
(unless you prefer to remain anonymous).

---

## Supported versions

Only the latest release receives security fixes.

---

## Security model

psst is a local-only tool. It does not make network requests and has no server component.

### What psst protects against

- **Shell history leakage** — secrets are never accepted as CLI flag values
- **Plaintext storage** — secure entries are encrypted at rest with AES-256-GCM
- **Weak key derivation** — keys are derived with Argon2id (64 MB memory, 3 iterations)
- **Accidental stdout leakage** — `psst get` copies to clipboard only, never prints

### What psst does NOT protect against

- **Root access** — a root user can read database and session files on the local machine
- **Memory forensics** — decrypted values exist briefly in process memory before being zeroed
- **Clipboard monitoring** — any process with clipboard access can read values after `psst get`
- **Unencrypted disk** — session cache (`~/.persist/.session`) should only be used with
  full-disk encryption enabled (e.g. FileVault on macOS)
- **Master password compromise** — there is no recovery mechanism if the master password is lost

### Known limitations

- `clipboard.WriteAll("")` may not reliably clear the clipboard on all platforms
- The string created by `string(plaintext)` when writing to the clipboard cannot be
  zeroed due to Go string immutability; it persists until garbage collected
