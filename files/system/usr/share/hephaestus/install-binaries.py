#!/usr/bin/env python3
"""Install only named executables from checksum-verified release archives."""
import argparse
import gzip
import hashlib
import io
import json
from pathlib import Path
import platform
import subprocess
import tarfile
import tempfile
import zipfile


def extract(data, url, binary):
    if url.endswith(".zip"):
        with zipfile.ZipFile(io.BytesIO(data)) as archive:
            members = [name for name in archive.namelist() if Path(name).name == binary]
            if len(members) != 1:
                raise ValueError(f"Expected one {binary}, found {members}")
            return archive.read(members[0])
    if url.endswith((".tar.gz", ".tgz")):
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
            members = [item for item in archive.getmembers() if item.isfile() and Path(item.name).name == binary]
            if len(members) != 1:
                raise ValueError(f"Expected one {binary}, found {[item.name for item in members]}")
            return archive.extractfile(members[0]).read()
    return gzip.decompress(data) if url.endswith(".gz") else data


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--destination", type=Path, default=Path("/usr/bin"))
    parser.add_argument("--verify-only", action="store_true")
    args = parser.parse_args()
    lock = json.loads(Path(__file__).with_name("binaries.lock.json").read_text())
    if not args.verify_only and (platform.system() != "Linux" or platform.machine() != lock["architecture"]):
        parser.error("These artifacts require Linux x86_64")
    if not args.verify_only:
        args.destination.mkdir(parents=True, exist_ok=True)
    for artifact in lock["artifacts"]:
        with tempfile.TemporaryFile() as download:
            subprocess.run(["curl", "-fsSL", "--retry", "3", artifact["url"]],
                           stdout=download, check=True)
            download.seek(0)
            data = download.read()
        if hashlib.sha256(data).hexdigest() != artifact["sha256"]:
            raise RuntimeError(f"Checksum mismatch: {artifact['url']}")
        for binary in artifact["binaries"]:
            content = extract(data, artifact["url"], binary)
            if content[:4] != b"\x7fELF" or content[18:20] != b"\x3e\x00":
                raise RuntimeError(f"Not an x86_64 ELF executable: {binary}")
            if not args.verify_only:
                target = args.destination / binary
                with target.open("xb") as output:
                    output.write(content)
                target.chmod(0o755)
            print(f"Verified {binary} ({artifact['version']})", flush=True)


if __name__ == "__main__":
    main()
