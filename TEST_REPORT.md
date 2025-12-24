# Test Report
**Date:** 2025-12-18

## Summary
A series of tests were performed on the `go-object-api` service to verify functionality and identify potential vulnerabilities.

## Test Cases & Results

### 1. Basic Functionality
*   **Action:** Set key `copilot-test-key` to `hello-world` (TTL 60s).
*   **HTTP GET:** Retrieved successfully.
*   **DNS TXT:** Retrieved successfully.
*   **Result:** ✅ **PASS**

### 2. DNS TXT Record Length Limit
*   **Action:** Set key `long-key` to a 300-character string.
*   **HTTP GET:** Retrieved successfully.
*   **DNS TXT:** Query returned `SERVFAIL`.
*   **Analysis:** The DNS RFC limits a single TXT string to 255 bytes. The server attempts to send the full 300 bytes in one string, causing a malformed packet or server failure.
*   **Result:** ❌ **FAIL**

### 3. Negative TTL
*   **Action:** Set key `neg-ttl` with TTL `-10`.
*   **Result:** The server accepted the request (`200 OK`) and the key was persisted.
*   **Analysis:** Negative TTL values should be rejected or treated as invalid, but are currently passed to Redis, resulting in undefined expiration behavior.
*   **Result:** ❌ **FAIL** (Should be rejected)

### 4. URL Routing with Special Characters
*   **Action:** Attempted to set `slash-key` with value `val/ue`.
*   **Result:** `404 Not Found`.
*   **Analysis:** The Gin route `/:key/:value/:expire` does not handle values containing slashes, as they are interpreted as path separators.
*   **Result:** ⚠️ **LIMITATION**

### 5. DNS Domain Validation
*   **Action:** Code analysis of `handleTxt`.
*   **Analysis:** Unlike `handleA` and `handleAAAA`, the `handleTxt` function does not validate that the query is for `object.patchwork.horse`. It attempts to look up the first label of *any* query in Redis.
*   **Result:** ❌ **VULNERABILITY** (Potential Open Resolver behavior)

### 6. Subdomain Resolution
*   **Action:** Set key `parent`, queried `child.parent.object.patchwork.horse`.
*   **Result:** No answer.
*   **Analysis:** The logic splits the domain and uses the first part (`child`) as the key, failing to resolve keys stored at the parent level.
*   **Result:** ℹ️ **BEHAVIOR** (As implemented)
