#!/usr/bin/env python3
"""Line-delimited JSON bridge between KNIRVENGINE and frida-python."""
import argparse
import json
import sys

# Frida 17's generated type annotations import NotRequired from typing even
# on Python 3.10, where the runtime symbol lives in typing_extensions. Keep
# the bridge compatible with the managed Ubuntu 22.04-era interpreter.
if not hasattr(__import__("typing"), "NotRequired"):
    import typing
    from typing_extensions import NotRequired

    typing.NotRequired = NotRequired

import frida


def emit(event):
    print(json.dumps(event), flush=True)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--pid", type=int, required=True)
    parser.add_argument("--script")
    parser.add_argument("--device", default="127.0.0.1:27042")
    options = parser.parse_args()
    # frida-server lives in the sandbox PID namespace. Connect to it remotely
    # from the host-side bridge so it receives a namespace PID, not a host PID.
    device = frida.get_device_manager().add_remote_device(options.device)
    session = device.attach(options.pid)
    current = None

    current_script_path = options.script

    def load(source):
        nonlocal current
        if current:
            current.unload()
        current = session.create_script(source)
        current.on("message", lambda message, data: emit({"type": "message", "message": message, "data": data.decode(errors="replace") if data else None}))
        current.load()
        emit({"type": "script_loaded"})

    def load_from_path(path):
        nonlocal current_script_path
        if not path:
            raise RuntimeError("no script path provided")
        with open(path, encoding="utf-8") as handle:
            load(handle.read())
        current_script_path = path

    if options.script:
        load_from_path(options.script)
    emit({"type": "attached", "pid": options.pid})
    for line in sys.stdin:
        try:
            command = json.loads(line)
            action = command.get("command")
            args = command.get("args") or {}
            if action == "load_script":
                load(args["source"])
            elif action == "reload":
                # The GUI edits a script path, not inline source (the browser
                # has no access to the sandbox's mounted filesystem), so
                # "reload" re-reads the path from disk rather than taking
                # source text directly like load_script does.
                load_from_path(args.get("script") or current_script_path)
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
