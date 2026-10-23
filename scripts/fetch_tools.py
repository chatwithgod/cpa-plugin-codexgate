"""Download pinned official CPA/LLVM-MinGW assets, verifying before extraction."""
import argparse
import os
from pathlib import Path, PurePosixPath
import shutil
import tarfile
import urllib.request
import zipfile
from release_common import ROOT, PINS, native_target, sha256


def download(url, name, expected):
    path = ROOT / ".cache/downloads" / name
    path.parent.mkdir(parents=True, exist_ok=True)
    if not path.exists():
        with urllib.request.urlopen(url, timeout=120) as source, path.open("xb") as target:
            shutil.copyfileobj(source, target)
    if sha256(path) != expected:
        raise RuntimeError(f"Checksum mismatch; inspect/remove only this download before retry: {path}")
    return path


def fetch_cpa():
    goos, goarch = native_target()
    target = f"{goos}-{goarch}"
    pin = PINS["cpa_assets"][target]
    url = f"https://github.com/router-for-me/CLIProxyAPI/releases/download/v{PINS['cpa']}/{pin['file']}"
    archive = download(url, pin["file"], pin["sha256"])
    output = ROOT / ".cache/cpa" / PINS["cpa"] / target
    output.mkdir(parents=True, exist_ok=True)
    name = "cli-proxy-api.exe" if goos == "windows" else "cli-proxy-api"
    if archive.suffix == ".zip":
        with zipfile.ZipFile(archive) as z:
            matches = [n for n in z.namelist() if PurePosixPath(n).name == name]
            if len(matches) != 1:
                raise ValueError("Expected exactly one CPA executable")
            data = z.read(matches[0])
    else:
        with tarfile.open(archive, "r:gz") as z:
            matches = [m for m in z.getmembers() if m.isfile() and PurePosixPath(m.name).name == name]
            if len(matches) != 1:
                raise ValueError("Expected exactly one CPA executable")
            with z.extractfile(matches[0]) as f:
                data = f.read()
    executable = output / name
    if executable.exists():
        if executable.read_bytes() != data:
            raise ValueError("Existing CPA cache differs from pinned archive")
    else:
        with executable.open("xb") as f:
            f.write(data)
        executable.chmod(0o755)
    return executable


def fetch_llvm():
    if native_target() != ("windows", "amd64"):
        raise ValueError("This LLVM-MinGW package is for Windows amd64")
    pin = PINS["llvm_mingw"]
    archive = download(f"https://github.com/mstorsjo/llvm-mingw/releases/download/{pin['version']}/{pin['file']}", pin["file"], pin["sha256"])
    destination = ROOT / ".cache/toolchains"
    destination.mkdir(parents=True, exist_ok=True)
    compiler = destination / pin["file"].removesuffix(".zip") / "bin/x86_64-w64-mingw32-gcc.exe"
    if not compiler.exists():
        with zipfile.ZipFile(archive) as z:
            for info in z.infolist():
                target = (destination / info.filename).resolve()
                if not target.is_relative_to(destination.resolve()):
                    raise ValueError("Unsafe toolchain archive path")
            z.extractall(destination)
    return compiler


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tool", choices=("cpa", "llvm-mingw"))
    parser.add_argument("--github-env")
    args = parser.parse_args()
    result = fetch_cpa() if args.tool == "cpa" else fetch_llvm()
    if args.github_env:
        with Path(os.environ["GITHUB_ENV"]).open("a", encoding="utf-8") as f:
            f.write(f"{args.github_env}={result}\n")
    print(result)
