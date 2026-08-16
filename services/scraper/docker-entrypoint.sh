#!/bin/sh
set -e
# Headed Chrome in Docker needs a virtual display (Qrator blocks headless).
if [ -f /.dockerenv ] && [ -z "${DISPLAY:-}" ]; then
  Xvfb :99 -screen 0 1920x1080x24 -nolisten tcp &
  export DISPLAY=:99
fi
exec "$@"
