#!/usr/bin/env python3
"""Fetch checksum-locked plugin sources; never decrypt workstation credentials."""
import argparse
import hashlib
import json
from pathlib import Path
import posixpath
import subprocess
import tarfile
import tempfile


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--destination", type=Path, default=Path("/usr/share/hephaestus/nvim/plugins"))
    args = parser.parse_args()
    assets = Path(__file__).resolve().parent
    sources = json.loads((assets / "sources.lock.json").read_text())["sources"]
    args.destination.mkdir(parents=True, exist_ok=True)
    for repository, source in sources.items():
        destination = args.destination / repository.rsplit("/", 1)[1]
        if destination.exists():
            raise RuntimeError(f"Refusing to overwrite existing sources: {destination}")
        with tempfile.TemporaryDirectory() as temporary:
            archive = Path(temporary) / "source.tar.gz"
            subprocess.run(
                ["curl", "-fsSL", "--retry", "3", source["url"], "-o", str(archive)],
                check=True,
            )
            if hashlib.sha256(archive.read_bytes()).hexdigest() != source["sha256"]:
                raise RuntimeError(f"Checksum mismatch: {repository}")
            with tarfile.open(archive) as contents:
                for member in contents.getmembers():
                    path = Path(member.name)
                    if path.is_absolute() or ".." in path.parts or member.isdev():
                        raise RuntimeError(f"Unsafe archive member: {member.name}")
                    if member.issym() or member.islnk():
                        target = posixpath.normpath(
                            posixpath.join(posixpath.dirname(member.name), member.linkname)
                            if member.issym() else member.linkname
                        )
                        if not target.startswith(path.parts[0] + "/"):
                            raise RuntimeError(f"Unsafe archive link: {member.name}")
            destination.mkdir()
            subprocess.run(
                ["tar", "-xzf", str(archive), "--strip-components=1", "-C", str(destination)],
                check=True,
            )
        print(f"Installed {repository} at {source['revision']}", flush=True)


if __name__ == "__main__":
    main()
