"""Native release build: pinned Go, tests, vet, race, then c-shared library."""
import argparse
import json
import os
import subprocess
from release_common import ROOT, PINS, LIBRARIES, native_target, sha256, version, write_new_json


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default="go")
    parser.add_argument("--cc", default=os.environ.get("CC", "gcc"))
    parser.add_argument("--version")
    args = parser.parse_args()
    release = version(args.version)
    goos, goarch = native_target()
    env = os.environ.copy()
    env.update({"CGO_ENABLED": "1", "GOTOOLCHAIN": "local", "GOOS": goos, "GOARCH": goarch, "CC": args.cc})
    env.setdefault("GOCACHE", str(ROOT / ".cache/go-build"))
    env.setdefault("GOPATH", str(ROOT / ".cache/gopath"))
    if goos == "darwin":
        env.setdefault("MACOSX_DEPLOYMENT_TARGET", PINS["macos_deployment_target"])
    go_version = subprocess.check_output([args.go, "version"], env=env, text=True).strip()
    if f"go{PINS['go']} " not in go_version:
        raise SystemExit(f"Required Go {PINS['go']}; got {go_version}")
    host_arch = subprocess.check_output([args.go, "env", "GOHOSTARCH"], env=env, text=True).strip()
    if host_arch != goarch:
        raise SystemExit("Release verification requires a native runner, not cross compilation")
    out = ROOT / "build" / release / f"{goos}-{goarch}"
    out.mkdir(parents=True, exist_ok=True)
    library = out / LIBRARIES[goos]
    if library.exists() or (out / "build-info.json").exists():
        raise SystemExit(f"Refusing to overwrite {out}")
    for command in (["test", "./..."], ["vet", "./..."], ["test", "-race", "./internal/..."]):
        subprocess.run([args.go, *command], cwd=ROOT, env=env, check=True)
    subprocess.run([args.go, "build", "-trimpath", "-buildvcs=false", "-ldflags",
                    f"-s -w -X codexgate/internal/plugin.Version={release}",
                    "-buildmode=c-shared", "-o", str(library), "./cmd/codexgate"], cwd=ROOT, env=env, check=True)
    compiler = subprocess.check_output([args.cc, "--version"], env=env, text=True).splitlines()[0]
    commit = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=ROOT, text=True).strip()
    dirty = bool(subprocess.check_output(["git", "status", "--porcelain"], cwd=ROOT, text=True).strip())
    info = {"version": release, "goos": goos, "goarch": goarch, "go": go_version,
            "compiler": compiler, "cpa": PINS["cpa"], "abi": 1, "schema": 6,
            "source_commit": commit, "source_dirty": dirty, "library": library.name,
            "library_sha256": sha256(library), "tests_passed": True, "vet_passed": True, "race_passed": True}
    write_new_json(out / "build-info.json", info)
    print(json.dumps(info, ensure_ascii=False))


if __name__ == "__main__":
    main()
