"""Build a store-compatible ZIP and checksum file without replacing any artifact."""
import argparse
import json
import zipfile
from release_common import ROOT, LIBRARIES, TARGETS, native_target, sha256, version


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version")
    parser.add_argument("--target", choices=TARGETS)
    args = parser.parse_args()
    release = version(args.version)
    target = args.target or "-".join(native_target())
    goos, goarch = target.split("-")
    build = ROOT / "build" / release / target
    info = json.loads((build / "build-info.json").read_text(encoding="utf-8"))
    library = build / LIBRARIES[goos]
    if info["version"] != release or info["library_sha256"] != sha256(library):
        raise SystemExit("Build metadata does not match the library")
    out = ROOT / "dist" / release / target
    out.mkdir(parents=True, exist_ok=True)
    archive = out / f"codexgate_{release}_{goos}_{goarch}.zip"
    if archive.exists() or (out / "checksums.txt").exists():
        raise SystemExit(f"Refusing to overwrite release artifacts in {out}")
    documents = ["README.md", "README.zh-CN.md", "LICENSE", "THIRD_PARTY_NOTICES.md", "CHANGELOG.md"]
    documents += [str(p.relative_to(ROOT)) for p in sorted((ROOT / "licenses").iterdir()) if p.is_file()]
    with zipfile.ZipFile(archive, "x", zipfile.ZIP_DEFLATED) as z:
        z.write(library, library.name)
        z.write(build / "build-info.json", "build-info.json")
        for name in documents:
            z.write(ROOT / name, name.replace("\\", "/"))
    with zipfile.ZipFile(archive) as z:
        assert z.testzip() is None
        libs = [n for n in z.namelist() if n.endswith((".dll", ".so", ".dylib"))]
        assert libs == [LIBRARIES[goos]]
    with (out / "checksums.txt").open("x", encoding="ascii", newline="\n") as f:
        f.write(f"{sha256(archive)}  {archive.name}\n")
    print(json.dumps({"archive": str(archive), "sha256": sha256(archive)}))


if __name__ == "__main__":
    main()
