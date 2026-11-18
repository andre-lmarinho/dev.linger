# DevLinger identity values. This file is the single source of truth for the
# AppID, the project URLs and the APT signing key fingerprint: every
# packaging script, GitHub workflow and the Go guard test
# (internal/app) read it from here. The packaging/linux asset filenames
# embed the AppID and must be renamed alongside any change to it.
APPID=io.github.devlinger.DevLinger

REPO_URL=https://github.com/andre-lmarinho/dev.linger
PAGES_URL=https://andre-lmarinho.github.io/dev.linger

# Fingerprint of the APT repository signing key; the installer refuses any
# other key.
GPG_FINGERPRINT=A17FD32EBCB92EC0C06CBA8E1B2A90F1DB7748F8
