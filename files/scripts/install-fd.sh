#!/usr/bin/env bash
set -euo pipefail

brew=/usr/share/homebrew/home/linuxbrew/.linuxbrew/bin/brew
staged_prefix=/usr/share/homebrew/home/linuxbrew/.linuxbrew
prefix_link=/home/linuxbrew/.linuxbrew
formula=/home/linuxbrew/.linuxbrew/opt/fd/bin/fd
destination=/usr/libexec/hephaestus/bin/fd
system_config=/etc/homebrew/brew.env
temporary=
created_linuxbrew_home=false
saved_system_config=false

if [[ ! -x "$brew" ]]; then
  echo "BlueBuild did not stage Homebrew in /usr/share/homebrew" >&2
  exit 1
fi

if [[ -e "$prefix_link" || -L "$prefix_link" ]]; then
  echo "Refusing to overwrite an existing Homebrew prefix: $prefix_link" >&2
  exit 1
fi

cleanup() {
  rm -f "$prefix_link"
  if [[ "$saved_system_config" == true ]]; then
    cp -p "$temporary/system-brew.env" "$system_config"
  fi
  if [[ -n "$temporary" ]]; then
    rm -rf "$temporary"
  fi
  if [[ "$created_linuxbrew_home" == true ]]; then
    rmdir /home/linuxbrew
  fi
}
trap cleanup EXIT

if [[ ! -d /home/linuxbrew ]]; then
  mkdir -p /home/linuxbrew
  created_linuxbrew_home=true
fi
ln -s "$staged_prefix" "$prefix_link"
temporary=$(mktemp -d)
mkdir -p "$temporary"/{cache,config/homebrew,logs,tmp}
# Homebrew loads brew.env after the process environment, so set build paths there too.
cat > "$temporary/config/homebrew/brew.env" <<EOF
HOMEBREW_CACHE=$temporary/cache
HOMEBREW_LOGS=$temporary/logs
HOMEBREW_TEMP=$temporary/tmp
EOF
chown -R 1000:1000 "$temporary"

# Secureblue reloads its system brew.env last; retain its policy, override only build paths.
if [[ -f "$system_config" ]]; then
  cp -p "$system_config" "$temporary/system-brew.env"
  saved_system_config=true
  printf '\n' >> "$system_config"
  cat "$temporary/config/homebrew/brew.env" >> "$system_config"
fi

for attempt in 1 2 3; do
  if setpriv --reuid=1000 --regid=1000 --clear-groups env \
  USER=linuxbrew \
  LOGNAME=linuxbrew \
  HOME="$temporary" \
  XDG_CACHE_HOME="$temporary/cache" \
  XDG_CONFIG_HOME="$temporary/config" \
  HOMEBREW_CACHE="$temporary/cache" \
  HOMEBREW_LOGS="$temporary/logs" \
  HOMEBREW_TEMP="$temporary/tmp" \
  HOMEBREW_NO_AUTO_UPDATE=1 \
  HOMEBREW_NO_INSTALL_CLEANUP=1 \
  HOMEBREW_NO_ANALYTICS=1 \
    "$brew" install fd; then
    break
  else
    status=$?
    if [[ "$attempt" == 3 ]]; then
      printf 'Homebrew fd installation failed after %s attempts (exit %s).\n' "$attempt" "$status" >&2
      exit "$status"
    fi
    printf 'Homebrew fd installation failed (exit %s); retrying attempt %s/3.\n' "$status" "$((attempt + 1))" >&2
    sleep "$((attempt * 5))"
  fi
done

if [[ ! -x "$formula" ]]; then
  echo "Homebrew installed fd but its executable is missing: $formula" >&2
  exit 1
fi

install -Dm755 "$formula" "$destination"
"$destination" --version
