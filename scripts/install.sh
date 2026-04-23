#!/usr/bin/env sh
# This script is used in the README and https://www.c3x.dev/docs/#quick-start
set -e

# check_sha is separated into a defined function so that we can
# capture the exit code effectively with `set -e` enabled
check_sha() {
  (
    cd /tmp/
    shasum -sc "$1"
  )

  return $?
}

os=$(uname | tr '[:upper:]' '[:lower:]')
arch=$(uname -m | tr '[:upper:]' '[:lower:]' | sed -e s/x86_64/amd64/)
if [ "$arch" = "aarch64" ]; then
  arch="arm64"
fi

version=${C3X_VERSION:-latest}
url="https://c3x.dev/downloads/${version}"
tar="c3x-$os-$arch.tar.gz"
echo "Downloading version ${version} of c3x-$os-$arch..."
curl -sL "$url/$tar" -o "/tmp/$tar"
echo

code=$(curl -s -L -o /dev/null -w "%{http_code}" "$url/$tar.sha256")
if [ "$code" = "404" ]; then
    echo "Skipping checksum validation as the sha for the release could not be found, no action needed."
else
  if [ -x "$(command -v shasum)" ]; then
    echo "Validating checksum for c3x-$os-$arch..."
    curl -sL "$url/$tar.sha256" -o "/tmp/$tar.sha256"

    if ! check_sha "$tar.sha256"; then
      echo
      read -r -p "Installation checksum failed. This could be a security issue. Would you like to continue? (y/n) " answer
      if [ "$answer" != "y" ]; then
        echo
        echo "Exiting, please email hello@c3x.dev for help."
        exit 1
      fi
    fi

    rm "/tmp/$tar.sha256"
  else
    echo "Skipping checksum validation as the shasum command could not be found, no action needed."
  fi
fi
echo

tar xzf "/tmp/$tar" -C /tmp
rm "/tmp/$tar"

echo "Moving /tmp/c3x-$os-$arch to /usr/local/bin/c3x (you might be asked for your password due to sudo)"
if [ -x "$(command -v sudo)" ]; then
  sudo mv "/tmp/c3x-$os-$arch" "/usr/local/bin/c3x"
else
  mv "/tmp/c3x-$os-$arch" "/usr/local/bin/c3x"
fi
echo
echo "Completed installing $(c3x --version)"
