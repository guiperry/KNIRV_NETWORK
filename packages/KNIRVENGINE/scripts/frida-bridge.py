#!/usr/bin/env python3
"""Line-delimited JSON bridge between KNIRVENGINE and frida-python."""
import argparse
import json
import sys

import frida


def emit(event):
    print(json.dumps(event), flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--pid", type=int, required=True)
    parser.add_argument("--script")
    options = parser.parse_args()
    device = frida.get_local_device()
    session = device.attach(options.pid)
    current = None

    def load(source):
        nonlocal current
        if current:
            current.unload()
        current = session.create_script(source)
        current.on("message", lambda message, data: emit({"type": "message", "message": message, "data": data.decode(errors="replace") if data else None}))
        current.load()
        emit({"type": "script_loaded"})

    if options.script:
        with open(options.script, encoding="utf-8") as handle:
            load(handle.read())
    emit({"type": "attached", "pid": options.pid})
    for line in sys.stdin:
        try:
            command = json.loads(line)
            action = command.get("command")
            args = command.get("args") or {}
            if action == "load_script":
                load(args["source"])
            elif action == "post":
                if not current:
                    raise RuntimeError("no script loaded")
                current.post(args.get("message"))
            elif action == "detach":
                break
            else:
                raise RuntimeError("unknown command: " + str(action))
        except Exception as error:
            emit({"type": "error", "error": str(error)})
    session.detach()
    emit({"type": "detached"})


if __name__ == "__main__":
    main()
