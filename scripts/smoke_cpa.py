"""Isolated CPA authentication smoke test. Synthetic logins; no model inference."""
from __future__ import annotations

import argparse
import base64
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
import re
from pathlib import Path
import secrets
import shutil
import socket
import subprocess
import time
import urllib.error
import urllib.request


def token(expires: int, nonce: str) -> str:
    def part(value):
        return base64.urlsafe_b64encode(json.dumps(value).encode()).decode().rstrip("=")
    return part({"alg": "RS256", "typ": "JWT"}) + "." + part({"exp": expires, "sub": nonce}) + ".c3ludGhldGlj"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--core", required=True)
    parser.add_argument("--library", "--dll", dest="library", required=True)
    parser.add_argument("--expected-version")
    parser.add_argument("--report")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    core, dll = Path(args.core).resolve(strict=True), Path(args.library).resolve(strict=True)
    run = root / ".runs" / (time.strftime("%Y%m%d-%H%M%S") + "-" + secrets.token_hex(3))
    run.mkdir(parents=True)
    isolated_plugins = run / "plugins"
    isolated_plugins.mkdir()
    shutil.copy2(dll, isolated_plugins / dll.name)
    home = run / "codex-home"
    home.mkdir()
    login = home / "auth.json"
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
    api_key, management_key = secrets.token_hex(24), secrets.token_hex(24)
    config = {
        "server": {"host": "127.0.0.1", "port": port},
        "management": {"allow-remote": False, "secret-key": management_key, "disable-control-panel": True},
        "access": {"api-keys": [api_key]},
        "oauth": {"auth-dir": str(run / "empty-upstream-accounts")},
        "observability": {"logs": {"debug": False, "logging-to-file": False, "request-log": False}, "usage": {"usage-statistics-enabled": False}},
        "plugins": {"enabled": True, "dir": str(isolated_plugins), "configs": {"codexgate": {"enabled": True}}},
    }
    config_path = run / "config.yaml"

    def save_config():
        temp = run / "config.yaml.tmp"
        temp.write_text(json.dumps(config), encoding="utf-8")
        os.replace(temp, config_path)

    def save_login(value):
        login.write_text(json.dumps({"auth_mode": "chatgpt", "tokens": {"access_token": value, "account_id": "synthetic-account"}}), encoding="utf-8")

    initial = token(int(time.time()) + 3600, "initial")
    rotated = token(int(time.time()) + 3600, "rotated")
    expired = token(int(time.time()) - 10, "expired")
    save_config()
    save_login(initial)
    protected = [Path.home() / ".codex" / "config.toml", Path.home() / ".codex" / "auth.json", core.parent / "config.yaml", core.parent.parent / "config.toml"]
    baseline = {p: hashlib.sha256(p.read_bytes()).hexdigest() for p in protected if p.is_file()}
    summary = {"kind": "isolated CPA auth, HTTP and WebSocket handshake; no model inference", "library_sha256": hashlib.sha256(dll.read_bytes()).hexdigest(), "core_sha256": hashlib.sha256(core.read_bytes()).hexdigest(), "protected_file_count": len(baseline), "checks": {}}
    base = f"http://127.0.0.1:{port}"

    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, *args, **kwargs):
            return None

    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirect())

    def request(path="/v1/models", bearer=None, method="GET", body=None):
        headers = {"Authorization": "Bearer " + bearer} if bearer else {}
        if body is not None:
            headers["Content-Type"] = "application/json"
        data = body if isinstance(body, bytes) else None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(base + path, headers=headers, method=method, data=data)
        try:
            with opener.open(req, timeout=5) as response:
                return response.status, response.read()
        except urllib.error.HTTPError as err:
            return err.code, err.read()

    def expect(name, expected, **kwargs):
        code, raw = request(**kwargs)
        if code != expected:
            raise AssertionError(f"{name}: expected HTTP {expected}, got {code}")
        summary["checks"][name] = code
        return raw

    def wait_for(check, label):
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            if proc.poll() is not None:
                raise RuntimeError("isolated CPA exited while waiting for " + label)
            try:
                if check():
                    return
            except (OSError, urllib.error.URLError):
                pass
            time.sleep(0.2)
        raise TimeoutError(label)

    def websocket_status(bearer=None):
        ws_key = base64.b64encode(secrets.token_bytes(16)).decode()
        lines = ["GET /v1/responses HTTP/1.1", f"Host: 127.0.0.1:{port}", "Upgrade: websocket", "Connection: Upgrade", f"Sec-WebSocket-Key: {ws_key}", "Sec-WebSocket-Version: 13"]
        if bearer:
            lines.append("Authorization: Bearer " + bearer)
        with socket.create_connection(("127.0.0.1", port), timeout=5) as sock:
            sock.sendall(("\r\n".join(lines) + "\r\n\r\n").encode())
            response = b""
            while b"\r\n\r\n" not in response and len(response) < 16384:
                chunk = sock.recv(4096)
                if not chunk:
                    break
                response += chunk
            code = int(response.split(b" ", 2)[1])
            if code == 101:
                sock.sendall(b"\x88\x80\x00\x00\x00\x00")
            return code

    env = os.environ.copy()
    env["CODEX_HOME"] = str(home)
    proc = None
    failure = None
    with (run / "core.log").open("wb") as log:
        try:
            proc = subprocess.Popen([str(core), "-config", str(config_path)], cwd=run, env=env, stdout=log, stderr=subprocess.STDOUT, creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
            print(json.dumps({"run": str(run), "pid": proc.pid, "phase": "isolated CPA"}), flush=True)
            wait_for(lambda: request("/v0/management/plugins", management_key)[0] == 200, "plugin registration")
            registration = json.loads(request("/v0/management/plugins", management_key)[1])
            (run / "plugin-registration.json").write_text(json.dumps(registration, indent=2), encoding="utf-8")
            loaded = [p for p in registration["plugins"] if p["id"] == "codexgate"]
            assert len(loaded) == 1 and loaded[0]["registered"] and loaded[0]["effective_enabled"], "native library registration"
            summary["checks"]["native_library_registered"] = True
            status = json.loads(expect("status", 200, path="/v0/management/codexgate/status", bearer=management_key))
            assert status["enabled"] and status["login"]["ready"] and not status["exclusive"]
            if args.expected_version:
                assert status["version"] == args.expected_version, "loaded plugin version"
            summary["plugin_version"] = status["version"]
            assert Path(status["auth_file"]) == login
            summary["checks"]["default_codex_home_detected"] = True
            expect("missing_credentials_rejected", 401)
            expect("unrelated_JWT_rejected", 401, bearer=token(int(time.time()) + 3600, "unrelated"))
            expect("original_api_key_kept", 200, bearer=api_key)
            expect("native_codex_credential_accepted", 200, bearer=initial)
            # Malformed JSON stops in the Images handler before any model call.
            # A 400 here proves authentication passed; missing credentials stay 401.
            for operation in ("generations", "edits"):
                image_request = {"path": f"/v1/images/{operation}", "method": "POST", "body": b"{"}
                expect(f"image_{operation}_missing_credentials", 401, **image_request)
                expect(f"image_{operation}_native_login_accepted", 400, bearer=initial, **image_request)
                expect(f"image_{operation}_api_key_kept", 400, bearer=api_key, **image_request)
            with ThreadPoolExecutor(max_workers=4) as pool:
                codes = list(pool.map(lambda _: request(bearer=initial)[0], range(12)))
            assert codes == [200] * 12, "concurrent native ABI authentication"
            summary["checks"]["concurrent_native_callbacks"] = True
            expect("management_stays_protected", 401, path="/v0/management/codexgate/status", bearer=initial)
            assert websocket_status() == 401
            assert websocket_status(initial) == 101
            summary["checks"]["websocket_auth_and_upgrade"] = True
            save_login(rotated)
            expect("old_token_revoked_on_rotation", 401, bearer=initial)
            expect("rotated_token_accepted", 200, bearer=rotated)
            save_login(expired)
            expect("expired_login_rejected", 401, bearer=expired)
            login.unlink()
            expect("logout_revokes_access", 401, bearer=rotated)
            expect("api_key_survives_logout", 200, bearer=api_key)
            login.write_text("{", encoding="utf-8")
            expect("invalid_file_fails_closed", 401, bearer=rotated)
            save_login(rotated)
            # Use the same supported management operation as ECPA, not a file
            # replacement that may remove a Windows file watch.
            expect("disable_plugin", 200, path="/v0/management/plugins/codexgate/enabled", bearer=management_key, method="PATCH", body={"enabled": False})
            wait_for(lambda: request(bearer=rotated)[0] == 401, "plugin disable revokes native credential")
            expect("api_key_survives_plugin_disable", 200, bearer=api_key)
            summary["checks"]["plugin_disable_revokes_native_credential"] = True
            expect("enable_plugin", 200, path="/v0/management/plugins/codexgate/enabled", bearer=management_key, method="PATCH", body={"enabled": True})
            wait_for(lambda: request(bearer=rotated)[0] == 200, "plugin re-enable")
            summary["checks"]["plugin_reenable"] = True
        except Exception as exc:
            failure = exc
            summary["failure"] = type(exc).__name__ + ": " + str(exc)
        finally:
            if proc is not None:
                proc.terminate()
                try:
                    proc.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait(timeout=5)
    log_text = (run / "core.log").read_text(encoding="utf-8", errors="replace")
    summary["checks"]["credentials_absent_from_logs"] = all(value not in log_text for value in [initial, rotated, expired, api_key, management_key])
    if failure is not None:
        safe_log = log_text
        for secret in [initial, rotated, expired, api_key, management_key, "synthetic-account"]:
            safe_log = safe_log.replace(secret, "<redacted>")
        safe_log = re.sub(r"\x1b\[[0-?]*[ -/]*[@-~]", "", safe_log)
        safe_log = "".join(c for c in safe_log if c in "\n\r\t" or ord(c) >= 32)
        summary["host_exit_code"] = None if proc is None else proc.returncode
        summary["host_failure_excerpt"] = safe_log[:5000] + ("\n...\n" + safe_log[-3000:] if len(safe_log) > 5000 else "")
    summary["checks"]["production_files_unchanged"] = all(p.is_file() and hashlib.sha256(p.read_bytes()).hexdigest() == value for p, value in baseline.items())
    summary["checks"]["isolated_process_stopped"] = proc is None or proc.poll() is not None
    summary["success"] = failure is None and all(summary["checks"].values())
    (run / "summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")
    if args.report:
        report = Path(args.report)
        report.parent.mkdir(parents=True, exist_ok=True)
        with report.open("x", encoding="utf-8") as f:
            json.dump(summary, f, ensure_ascii=False, indent=2)
    print(json.dumps(summary, ensure_ascii=False), flush=True)
    if not summary["success"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
