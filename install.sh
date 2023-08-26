#!/bin/sh

TMP_NAME="/tmp/$(head -n 1 -c 32 /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32)"
PRERELEASE=false
if [ "$1" = "--prerelease" ]; then
	PRERELEASE=true
fi

if which curl >/dev/null; then
	if curl --help 2>&1 | grep "--progress-bar" >/dev/null 2>&1; then
		PROGRESS="--progress-bar"
	fi

	set -- curl -L $PROGRESS -o "$TMP_NAME"
	if [ "$PRERELEASE" = true ]; then
		LATEST=$(curl -sL https://api.github.com/repos/tez-capital/tezbake/releases | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g' | head -n 1 | tr -d '[:space:]')
	else
		LATEST=$(curl -sL https://api.github.com/repos/tez-capital/tezbake/releases/latest | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g' | tr -d '[:space:]')
	fi
else
	if wget --help 2>&1 | grep "--show-progress" >/dev/null 2>&1; then
		PROGRESS="--show-progress"
	fi
	set -- wget -q $PROGRESS -O "$TMP_NAME"
	if [ "$PRERELEASE" = true ]; then
		LATEST=$(wget -qO- https://api.github.com/repos/tez-capital/tezbake/releases | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g' | head -n 1 | tr -d '[:space:]')
	else
		LATEST=$(wget -qO- https://api.github.com/repos/tez-capital/tezbake/releases/latest | grep tag_name | sed 's/  "tag_name": "//g' | sed 's/",//g' | tr -d '[:space:]')
	fi
fi

if tezbake version | grep "$LATEST"; then
	echo "Latest tezbake already available."
	exit 0
fi

PLATFORM=$(uname -m)
if [ "$PLATFORM" = "x86_64" ]; then
	PLATFORM="amd64"
elif [ "$PLATFORM" = "aarch64" ]; then
	PLATFORM="arm64"
else
	echo "Unsupported platform: $PLATFORM" 1>&2
	exit 1
fi

if [ "$PRERELEASE" = true ]; then
	echo "Downloading latest tezbake prerelease for $PLATFORM..."
else
	echo "Downloading tezbake-linux-$PLATFORM $LATEST..."
fi

if "$@" "https://github.com/tez-capital/tezbake/releases/download/$LATEST/tezbake-linux-$PLATFORM" &&
	mv "$TMP_NAME" /usr/sbin/tezbake &&
	chmod +x /usr/sbin/tezbake; then
	if [ "$1" = "--prerelease" ]; then
		echo "Latest tezbake prerelease for $PLATFORM successfully installed."
	else
		echo "tezbake $LATEST for $PLATFORM successfully installed."
	fi
else
	echo "tezbake installation failed!" 1>&2
	exit 1
fi
