## SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
## SPDX-License-Identifier: Apache-2.0
#!/usr/bin/env sh
set -e

# check arguments for an option that would cause /argus to stop
# return true if there is one
_want_help() {
    local arg
    for arg; do
        case "$arg" in
            -'?'|--help|-v)
                return 0
                ;;
        esac
    done
    return 1
}

_main() {
    # if command starts with an option, prepend argus
    if [ "${1:0:1}" = '-' ]; then
        set -- /argus "$@"
    fi

    # skip setup if they aren't running /argus or want an option that stops /argus
    if [ "$1" = '/argus' ] && ! _want_help "$@"; then
        echo "Entrypoint script for argus Server ${VERSION} started."

        if [ ! -s /etc/argus/argus.yaml ]; then
            echo "Building out template for file"
            /bin/spruce merge /tmp/argus_spruce.yaml > /etc/argus/argus.yaml
        fi
    fi

    exec "$@"
}

_main "$@"
