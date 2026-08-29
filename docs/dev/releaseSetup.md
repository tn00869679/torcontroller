# Release Pipeline 設定

本專案 CI/CD 的一次性初始設定。以下沒有任何東西會存進 repo — 每一項都放在你的
GitHub 帳號或 repository secrets 裡。

Repository：`tn00869679/torcontroller`

---

## 1. GPG 簽章金鑰

`.github/workflows/release.yml` 透過 `scripts/build_and_sign.sh` 對 `.deb` 簽章。
該腳本是 `set -e`，找不到金鑰會直接中止，所以**沒有這把金鑰就無法發布 release**。

> **在 WSL / Linux shell 裡執行，不要用 PowerShell。** Windows PowerShell 的 `>`
> 重導向會寫出帶 BOM 的檔案，那三個位元組會讓 CI 端的 `gpg --batch --import`
> 以 `no valid OpenPGP data found` 失敗 —— 而檔案肉眼看起來完全正常。
> `shred` 在 Windows 也不存在。

```bash
# gpg 2.x 需要 pinentry 才能輸入 passphrase，而 pinentry-curses 在視窗太小時
# 會以 "Screen or window too small - skipped" 中止。改走 loopback 讓 gpg 直接
# 在終端機行內詢問，繞開對話框：
echo "allow-loopback-pinentry" >> ~/.gnupg/gpg-agent.conf
gpgconf --kill gpg-agent
export GPG_TTY=$(tty)

# 產生金鑰 — 選 RSA 4096。必須設 passphrase（腳本會餵一組進去）。
gpg --pinentry-mode loopback --full-generate-key

# 查出 key ID
gpg --list-secret-keys --keyid-format=long
#   sec   rsa4096/ABCD1234EF567890 2026-08-02 [SC]
#         6032ECF9AC84418EB1FDDF70ABCD1234EF567890
#         <-- long key ID 是 ABCD1234EF567890，底下那行是完整指紋，兩者都可用

# 以 ASCII armor 格式匯出私鑰。
# 注意輸出路徑在 repo 之外 — 見下方警告。
gpg --armor --pinentry-mode loopback --export-secret-keys ABCD1234EF567890 > ~/torcontroller-gpg-private.asc
```

匯出這一步會提示 `Enter passphrase:`，**同時也是密碼正確與否的唯一驗證機會** ——
密碼錯就會失敗。若當初是在壞掉的 pinentry 裡輸入、已記不得，直接砍掉重產：

```bash
gpg --batch --yes --delete-secret-keys <KEYID>
gpg --batch --yes --delete-keys <KEYID>
```

> **絕對不要把私鑰匯出到這個 repo 的目錄裡。** 本 repo 是公開的。
> `.gitignore` 已經擋掉 `*.asc` / `.env` / `*.key` 等樣式，但別依賴它當唯一防線 —
> 匯出到 `~` 底下，貼進 GitHub secret 之後立刻刪除。
> （`release.yml` 會在 CI 執行期間自行於 workspace 產生 `.env` 與 `private-key.asc`，
> 那是 runner 上的暫時檔案，跟你本機無關。）

### 把私鑰貼進 GitHub

私鑰有上百行 base64。**不要用滑鼠從終端機拖曳選取** —— 視窗較窄時長行會折行顯示，
某些選取模式會把折行變成真的換行字元，貼出來的金鑰就壞了，而且不會有任何提示，
要等 CI 跑 release 才爆。從 WSL 直接送進 Windows 剪貼簿：

```bash
cat ~/torcontroller-gpg-private.asc | clip.exe
```

沒有輸出就是成功。到 GitHub 的 secret 欄位 Ctrl+V 即可。貼完清掉剪貼簿：

```bash
echo -n "" | clip.exe
```

到 Settings → Secrets and variables → Actions 新增三個 repository secret：

| Secret | 內容 |
|---|---|
| `GPG_PRIVATE_KEY` | `~/torcontroller-gpg-private.asc` 的完整內容，含頭尾的 `-----BEGIN/END-----` 行 |
| `GPG_PASSPHRASE` | 你設定的 passphrase |
| `GPG_PUBLIC_KEY` | **key ID / fingerprint**，例如 `ABCD1234EF567890` |

> **陷阱**：名字雖然叫 `GPG_PUBLIC_KEY`，但它**不是** armored 公鑰。
> `build_and_sign.sh` 有兩處用到它 — `gpg --list-keys \| grep -q "$GPG_PUBLIC_KEY"`
> 以及 `dpkg-buildpackage -k"$GPG_PUBLIC_KEY"` — 兩者要的都是金鑰識別碼。
> 貼 armored 區塊進去會卡在 grep 那一關直接失敗。

三個 secret 都存好之後，立刻刪掉本機的匯出檔：

```bash
shred -u ~/torcontroller-gpg-private.asc 2>/dev/null || rm -P ~/torcontroller-gpg-private.asc 2>/dev/null || rm ~/torcontroller-gpg-private.asc
```

私鑰仍在你的 GPG keyring 裡（`gpg --list-secret-keys` 查得到），刪掉的只是匯出的副本。

---

## 2. GHCR container images

CI 會拉兩個 image。它們必須存在於**你的** namespace 底下，而且 Actions 讀得到。

先建立 PAT（Settings → Developer settings → Personal access tokens → classic），
scope 勾 `write:packages`（會自動含 `read:packages`）。

```bash
export CR_PAT=<your token>
echo "$CR_PAT" | docker login ghcr.io -u tn00869679 --password-stdin

docker buildx create --use   # 若還沒有 builder，執行一次即可

docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/tn00869679/torcontroller/torcontroller-build:dev \
  -f dockerfile.build . --push

docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/tn00869679/torcontroller/torcontroller-test-env:dev \
  -f dockerfile.testenv . --push
```

推完之後，到 `https://github.com/users/tn00869679/packages`，**兩個 package 都要**：

1. Package settings → **Change visibility → Public**。
2. Package settings → **Manage Actions access** → 加入 `torcontroller` repository，
   權限至少 `Read`。

> **陷阱**：GHCR package **預設是 private**。`test.yml` 把 test image 宣告成
> job 層級的 `container:`，而且完全沒有 `docker login` 步驟 — image 只要是 private，
> 每次 CI 都會在跑到第一個 step 之前就掛掉。上面第 1 點不是可選項。

> **建議**：`:dev` 是 mutable tag。同時多推一個不可變的 tag
>（`-t ghcr.io/.../torcontroller-build:2026-08-02`）並讓 workflow 釘在那個 tag 上，
> 這樣日後重建 `:dev` 才不會在你不知情的狀況下改變 CI 行為。

---

## 3. Release token

`release.yml` 用 `secrets.PAT_TOKEN` 來建立 GitHub Release 並上傳檔案。
建立一個 scope 為 `repo` 的 classic PAT，存成 `PAT_TOKEN`。

同一個 repo 的 release 其實用內建的 `GITHUB_TOKEN` 也可以；但這份 workflow 當初
是照 PAT 寫的，除非你要改 workflow，否則就沿用 PAT 保持一致。

---

## 4. Codecov（選用，不影響 CI 成敗）

`README.md` / `READMEJP.md` 裡的 badge 指向
`codecov.io/gh/tn00869679/torcontroller`，在你啟用之前都會是破圖。

1. 到 <https://codecov.io> 用 GitHub 登入，把這個 repository 加進去。
2. 把 upload token 存成 `CODECOV_TOKEN` repository secret。
3. 在 `.github/workflows/test.yml` 把 token 傳給 action：

```yaml
    - name: Upload coverage reports to Codecov
      uses: codecov/codecov-action@v5
      continue-on-error: true
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
```

coverage 上傳失敗不會讓 test job 失敗。

---

## 5. 發布第一個 release

v1.1.0 已經發布，兩份 README 的下載連結目前有效。以下是**下一次**發版的流程。
每次發版都要讓 changelog 版本、`cmd/version.go` 的字串、git tag 與 README 網址四者一致。

```bash
# 1. 在 Debian changelog 記錄這次發布。trailer 必須是「你」—
#    既有的 entry 屬於原作者，不要動。
#    （dch 在 devscripts 套件裡。）
dch --newversion 1.2.0 --distribution unstable "Describe your changes"

# 2. 同步 CLI 字串 — cmd/version.go 硬編碼了版本號兩次，
#    cmd/version_test.go 也對它做斷言。
#    三處都改完之後：
go test ./cmd/...

# 3. 打 tag 並推上去 — 這才是觸發 release.yml 的動作。
git tag v1.2.0
git push origin v1.2.0
```

記得把 `README.md` 和 `READMEJP.md` 裡的 `wget` 網址改成新版本號。

### 試跑整條 pipeline

`dockerfile.build` 裡記載了一個拋棄式 tag，可以在不發布正式版本的情況下測 CI：

```bash
git tag v.dev && git push origin v.dev
# ... 到 Actions 頁面檢查執行結果 ...
git tag -d v.dev && git push origin --delete v.dev
```

---

## 6. 本機跑測試

單元測試**在 Windows 上無法編譯** — `initializer/sudoersVerify.go` 用到
`syscall.Stat_t`，那是 Unix-only 的型別。請用 WSL、Linux 機器，或 test-env container：

```bash
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o /tmp/torcontroller .
go test ./...
```

---

## Upstream

`upstream` remote 用來追蹤原專案，方便 diff 和 cherry-pick：

```bash
git fetch upstream
git log --oneline upstream/main
```

它被刻意設定成 `tagOpt = --no-tags`：否則上游的 `v*` tag 會被抓進本地，
之後一個 `git push --tags` 就會拿「不在本 fork 歷史中的 commit」觸發 `release.yml`。
