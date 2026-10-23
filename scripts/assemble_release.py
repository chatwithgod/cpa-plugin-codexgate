"""Require five successful native jobs and assemble immutable release assets."""
import argparse
import json
import os
from pathlib import Path
import shutil
import zipfile
from release_common import ROOT, TARGETS, LIBRARIES, sha256, version, write_new_json


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifacts", type=Path, required=True)
    parser.add_argument("--tag", required=True)
    args = parser.parse_args()
    release = version()
    if args.tag != "v" + release:
        raise SystemExit("Release tag and VERSION disagree")
    out = ROOT / "dist/release" / release
    out.mkdir(parents=True, exist_ok=False)
    evidence = {"version": release, "cpa": "8.0.38", "source_commit": os.environ["GITHUB_SHA"], "platforms": {}}
    hashes = []
    for target in TARGETS:
        goos, goarch = target.split("-")
        name = f"codexgate_{release}_{goos}_{goarch}.zip"
        matches = list(args.artifacts.rglob(name))
        if len(matches) != 1:
            raise ValueError(f"Expected one artifact for {target}")
        archive = matches[0]
        folder = archive.parent
        build = json.loads((folder / "build-info.json").read_text(encoding="utf-8"))
        smoke = json.loads((folder / "smoke.json").read_text(encoding="utf-8"))
        store = json.loads((folder / "store-install.json").read_text(encoding="utf-8"))
        assert build["source_commit"] == os.environ["GITHUB_SHA"] and not build["source_dirty"]
        assert build["version"] == release and (build["goos"], build["goarch"]) == (goos, goarch)
        assert all(build[k] for k in ("tests_passed", "vet_passed", "race_passed"))
        assert smoke["success"] and smoke["plugin_version"] == release
        assert smoke["library_sha256"] == build["library_sha256"]
        assert store["success"] and store["checksum_mismatch_rejected"] and not store["overwritten"]
        digest = sha256(archive)
        assert (folder / "checksums.txt").read_text(encoding="ascii").strip() == f"{digest}  {name}"
        with zipfile.ZipFile(archive) as z:
            assert z.testzip() is None
            assert [n for n in z.namelist() if n.endswith((".dll", ".so", ".dylib"))] == [LIBRARIES[goos]]
            assert json.loads(z.read("build-info.json")) == build
        record = {"build": build, "host_smoke": smoke, "store_installer": store}
        if goos == "linux":
            runtime = json.loads((folder / "runtime-compatibility.json").read_text(encoding="utf-8"))
            assert runtime["passed"] and runtime["baseline"] == "2.17"
            record["runtime_compatibility"] = runtime
        evidence["platforms"][target] = record
        shutil.copy2(archive, out / name)
        hashes.append(f"{digest}  {name}\n")
    with (out / "checksums.txt").open("x", encoding="ascii", newline="\n") as f:
        f.writelines(sorted(hashes))
    write_new_json(out / "verification.json", evidence)
    print(out)


if __name__ == "__main__":
    main()
