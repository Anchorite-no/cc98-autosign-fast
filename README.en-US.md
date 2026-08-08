

# CC98 Auto Sign-in Script / cc98-autosign-fast

A lightweight auto sign-in tool for Zhejiang University's CC98, and a ready-to-run CC98 auto sign-in script.
Supports WebVPN, multiple accounts, Windows, and Linux.

If you are looking for a ready-to-use CC98 sign-in script, this repository provides the corresponding open-source implementation.

English: A fast CC98 auto sign-in tool with a lightweight WebVPN flow and prebuilt Go releases.

The current main implementation is in Go, and the release packages provide:

- Windows: `cc98-autosign-fast.exe` + `.env`
- Linux: `cc98-autosign-fast` + `.env`

## Usage

### Windows

1. Download the Windows release package from GitHub Releases
2. Extract and fill in the `.env` file
3. Double-click `cc98-autosign-fast.exe`
4. The program will automatically sign in and display the results in the window

### Linux

1. Download the Linux release package from GitHub Releases
2. Extract and fill in the `.env` file
3. Run `./cc98-autosign-fast` in the terminal
4. The program will output the sign-in results and exit directly

If `.env` does not exist, the program will automatically generate a template and prompt you to fill it in before running again.

## `.env` Multi-Account Example

```env
WEBVPN_USER=Your WebVPN Username
WEBVPN_PASS=Your WebVPN Password

CC98_ACCOUNT_COUNT=2

CC98_USER_1=First CC98 Username
CC98_PASS_1=First CC98 Password

CC98_USER_2=Second CC98 Username
CC98_PASS_2=Second CC98 Password
```

If you have more accounts, continue adding them below:

```env
CC98_USER_3=Third CC98 Username
CC98_PASS_3=Third CC98 Password
```

## Output Example

```text
Account 1 (anchorite) Sign-in successful · 1141 Wealth Points · 30 consecutive days
Account 2 (example2) Sign-in successful · 1155 Wealth Points · 5 consecutive days
```

## Request Chain

The current implementation splits requests into two parts: the WebVPN authentication chain and the CC98 business chain.

### WebVPN Authentication Chain

During a cold start, the WebVPN part only performs the minimal login flow:

1. `GET /login`
2. `POST /do-login`
3. Only performs `POST /do-confirm-login` if `NEED_CONFIRM` is returned

During a hot start, it first reads `.webvpn-cookie-cache.json` in the working directory. If the cache is valid, the entire WebVPN login process is skipped directly.

If the `token` request for the first account is redirected back to the WebVPN login page, the program will:

1. Clear the current cookie
2. Re-execute the WebVPN authentication chain
3. Retry only the current account once

### CC98 Business Chain

After the WebVPN session is established, each account goes through the same business chain:

1. `POST connect/token`
2. `POST me/signin`
3. `GET me/signin`

Where:

- `connect/token` is used to exchange an access token for the current account
- `me/signin` is used to execute the sign-in
- `GET me/signin` is used to supplement reading the consecutive sign-in days and today's reward

## Why It's Fast

### Why the WebVPN Part is Fast

- The WebVPN flow only retains the minimal login chain, without navigating to the homepage
- No longer requests `user/info`
- After hitting the cookie cache, the entire WebVPN login is skipped directly

### Why the CC98 Part is Fast

- No dynamic host rewriting; directly uses fixed `connect/token` and `me/signin` routes
- Multiple accounts share the same WebVPN session, avoiding repeated logins for each account
- During a hot start, only the actual business requests remain

Therefore, during a cold start, the main time consumption is the WebVPN login. During a hot start, the time is mainly spent on the CC98 token / signin / sign-info steps.

## Fixed Paths and Caching Mechanism

- The current implementation relies on verified fixed WebVPN token/sign routes
- The program writes `.webvpn-cookie-cache.json` in the working directory
- The truly critical element in the cache is `wengine_vpn_ticketwebvpn_zju_edu_cn`
- `route` uses the fixed default value built into the program

This is also why the current implementation achieves a noticeable speed improvement after hitting the cache.

## Repository Structure

- `src/`
  - Go main implementation and tests
- `python-reference/`
  - Current Python reference implementation, used only for protocol reference and debugging
- `.github/workflows/release.yml`
  - GitHub Actions for automatic building and releasing

## Local Build

Windows:

```powershell
./build-release.ps1
```

Linux:

```bash
bash ./build-release.sh
```

After building, the artifacts will be generated in the local `dist/` directory, but this directory is not tracked by git.

## Notes

- Do not commit `.env`, `.webvpn-cookie-cache.json`, `dist/`, or any real account passwords
- `python-reference/` is not the main release entry; regular users should prioritize using the Go release
