# Release Pipeline Setup

One-time bootstrap for this fork's CI/CD. Nothing here is stored in the repo —
every item below lives in your GitHub account or repository secrets.

Repository: `tn00869679/torcontroller`

---

## 1. GPG signing key

`.github/workflows/release.yml` signs the `.deb` via `scripts/build_and_sign.sh`.
The script hard-fails (`set -e`) if the key is absent, so the release job cannot
run without this.

```bash
# Generate — choose RSA 4096. A passphrase is required (the script feeds one).
gpg --full-generate-key

# Find the key ID
gpg --list-secret-keys --keyid-format=long
#   sec   rsa4096/ABCD1234EF567890 2026-08-02 [SC]
#         <-- long key ID is ABCD1234EF567890

# Export the private key in ASCII armor
gpg --armor --export-secret-keys ABCD1234EF567890 > private-key.asc
```

Add three repository secrets (Settings → Secrets and variables → Actions):

| Secret | Value |
|---|---|
| `GPG_PRIVATE_KEY` | full contents of `private-key.asc`, including the `-----BEGIN/END-----` lines |
| `GPG_PASSPHRASE` | the passphrase you chose |
| `GPG_PUBLIC_KEY` | **the key ID / fingerprint**, e.g. `ABCD1234EF567890` |

> **Gotcha:** despite the name, `GPG_PUBLIC_KEY` is *not* an armored public key.
> `build_and_sign.sh` uses it two ways — `gpg --list-keys \| grep -q "$GPG_PUBLIC_KEY"`
> and `dpkg-buildpackage -k"$GPG_PUBLIC_KEY"` — both of which want a key identifier.
> Pasting an armored block here makes the job fail at the grep check.

Delete `private-key.asc` from disk once the secret is saved.

---

## 2. GHCR container images

CI pulls two images. They must exist under **your** namespace and be readable by
Actions.

Create a PAT (Settings → Developer settings → Personal access tokens → classic)
with scope `write:packages` (which implies `read:packages`).

```bash
export CR_PAT=<your token>
echo "$CR_PAT" | docker login ghcr.io -u tn00869679 --password-stdin

docker buildx create --use   # once, if you have no builder yet

docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/tn00869679/torcontroller/torcontroller-build:dev \
  -f dockerfile.build . --push

docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/tn00869679/torcontroller/torcontroller-test-env:dev \
  -f dockerfile.testenv . --push
```

Then, **for each package** at `https://github.com/users/tn00869679/packages`:

1. Package settings → **Change visibility → Public**.
2. Package settings → **Manage Actions access** → add the `torcontroller`
   repository with at least `Read` role.

> **Gotcha:** GHCR packages are **private by default**. `test.yml` declares the
> test image as a job-level `container:` and performs no `docker login`, so a
> private image makes every CI run fail before the first step. Step 1 above is
> not optional.

> **Recommended:** `:dev` is a mutable tag. Also push an immutable one
> (`-t ghcr.io/.../torcontroller-build:2026-08-02`) and pin the workflows to it,
> so a future rebuild of `:dev` cannot silently change CI behaviour underneath you.

---

## 3. Release token

`release.yml` uses `secrets.PAT_TOKEN` to create the GitHub Release and upload
assets. Create a classic PAT with the `repo` scope and store it as `PAT_TOKEN`.

The built-in `GITHUB_TOKEN` would also work for same-repo releases; the workflow
was written against a PAT, so keep it consistent unless you edit the workflow.

---

## 4. Codecov (optional, non-blocking)

The badges in `README.md` / `READMEJP.md` point at
`codecov.io/gh/tn00869679/torcontroller` and stay broken until you enable it.

1. Sign in at <https://codecov.io> with GitHub and add the repository.
2. Copy the upload token into the `CODECOV_TOKEN` repository secret.
3. Pass it to the action in `.github/workflows/test.yml`:

```yaml
    - name: Upload coverage reports to Codecov
      uses: codecov/codecov-action@v5
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
```

Coverage upload failure does not fail the test job.

---

## 5. Cutting the first release

The `.deb` download links in both READMEs point at `v1.1.0` of **this** repo.
They are dead until you publish that tag.

```bash
# 1. Record the release in Debian's changelog. The trailer must be YOU —
#    existing entries belong to the original author and stay untouched.
#    (dch is in the devscripts package.)
dch --newversion 1.2.0 --distribution unstable "Describe your changes"

# 2. Keep the CLI string in sync — cmd/version.go hardcodes the version twice,
#    and cmd/version_test.go asserts on it.
#    Update all three, then:
go test ./cmd/...

# 3. Tag and push — this is what fires release.yml.
git tag v1.2.0
git push origin v1.2.0
```

Update the `wget` URLs in `README.md` and `READMEJP.md` to the new version.

### Dry-running the pipeline

`dockerfile.build` documents a throwaway tag for testing CI without publishing a
real version:

```bash
git tag v.dev && git push origin v.dev
# ... inspect the Actions run ...
git tag -d v.dev && git push origin --delete v.dev
```

---

## 6. Local test runs

Unit tests cannot compile on Windows — `initializer/sudoersVerify.go` uses
`syscall.Stat_t`, which is Unix-only. Use WSL, a Linux box, or the test-env
container:

```bash
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /tmp/torcontroller .
go test ./...
```

---

## Upstream

`upstream` remote tracks the original project for diffing and cherry-picking:

```bash
git fetch upstream
git log --oneline upstream/main
```

It is configured with `tagOpt = --no-tags` on purpose: upstream's `v*` tags would
otherwise land locally and a `git push --tags` would fire `release.yml` against
commits that are not in this fork's history.
