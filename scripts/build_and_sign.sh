#!/bin/bash
set -e

# Load environment variables from .env file
if [ -f .env ]; then
  export $(cat .env | xargs)
else
  echo ".env file not found. Exiting."
  exit 1
fi

# Install Go
GO_VERSION=1.21.13
curl -LO https://golang.org/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz
tar -C /usr/local -xzf go${GO_VERSION}.linux-${ARCH}.tar.gz
rm go${GO_VERSION}.linux-${ARCH}.tar.gz

export PATH="/usr/local/go/bin:${PATH}"
go version # Verify installation

export GPG_TTY=$(tty)

# Confirm the existence of the private key file
if [ ! -f ./private-key.asc ]; then
    echo "Error: private-key.asc not found."
    exit 1
fi

gpg --batch --import ./private-key.asc

# Check if the key was imported successfully.
if ! gpg --list-keys | grep -q "$GPG_PUBLIC_KEY"; then
    echo "Error: Public key $GPG_PUBLIC_KEY not found."
    exit 1
fi

echo "allow-loopback-pinentry" >> ~/.gnupg/gpg-agent.conf
echo "default-cache-ttl 600" >> ~/.gnupg/gpg-agent.conf
echo "max-cache-ttl 7200" >> ~/.gnupg/gpg-agent.conf
gpgconf --kill gpg-agent
gpgconf --launch gpg-agent

# Ensure GPG Agent is activated successfully
if ! pgrep -x "gpg-agent" > /dev/null; then
    echo "Error: GPG Agent did not start successfully."
    exit 1
fi

echo "$GPG_PASSPHRASE" | gpg --batch --yes --passphrase-fd 0 --pinentry-mode loopback -o /tmp/signed_dummy_file --sign /etc/hostname

dpkg-buildpackage -k"$GPG_PUBLIC_KEY" -a"$ARCH"

# Write the name of the resulting .deb file to GITHUB_ENV
deb_file=$(ls ../torcontroller_*_"$ARCH".deb)
if [ -z "$deb_file" ]; then
    echo "Error: .deb file not found."
    exit 1
fi

# Extract only the file name (basename)
deb_file_name=$(basename "$deb_file")
echo "deb_file_name=$deb_file_name" >> /workspace/container_up_env

dpkg-sig --sign builder --gpg-options="--pinentry-mode loopback" "$deb_file"