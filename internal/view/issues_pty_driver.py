"""Exercise IssueList.Render with a real controlling PTY, using synthetic data only.

Usage: python3 issues_pty_driver.py [--timeout 35] -- COMMAND [ARG ...]
COMMAND is normally a Go view test binary with
-test.run=^TestIssueListRenderPTYHelper$ -test.timeout=30s. An explicit container
run -it command may replace it; no container or VM is started by default.
The child environment is allowlisted, with GLAMOUR_STYLE and CI unset.
Exit codes: 0 passes, 1 regression/operational failure, 2 invalid arguments.
"""

import argparse
import codecs
import errno
import fcntl
import os
import pty
import re
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time
import unicodedata


class Terminal:
    """Track tcell's 120x40 display and answer OSC 10/11 and cursor DSR.

    This is not a general terminal emulator: CSI cursor addressing/movement,
    erase, SGR/modes, charset selection, BEL/ST OSC and split UTF-8/escapes
    cover this fixture. Unsupported CSI fails rather than faking readiness.
    """

    def __init__(self, fd: int) -> None:
        self.fd = fd
        self.cells = [[" "] * 120 for _ in range(40)]
        self.row = self.col = 0
        self.pending = ""
        self.decoder = codecs.getincrementaldecoder("utf-8")("replace")
        self.background_queries = 0
        self.active = False
        self.manual_slash = False
        self.transcript = ""

    def send(self, data: bytes) -> None:
        while data:
            count = os.write(self.fd, data)
            if count == 0:
                raise RuntimeError("PTY write returned zero")
            data = data[count:]

    def text(self) -> str:
        return "\n".join("".join(row) for row in self.cells)

    def csi(self, sequence: str) -> None:
        final, args = sequence[-1], sequence[2:-1]
        if sequence == "\x1b[6n":
            self.send(b"\x1b[1;1R")
            return
        if final in "hl" and args == "?1049":
            if final == "h":
                if not self.background_queries:
                    raise RuntimeError("TUI started without a preflight OSC 11 exchange (notty/CI/forced style?)")
                self.active = True
            self.cells = [[" "] * 120 for _ in range(40)]
            self.row = self.col = 0
            return
        if final in "mhlrtq":
            return
        values = [int(value or "0") for value in args.split(";")]
        value = values[0] or 1
        if final in "Hf":
            self.row = value - 1
            self.col = (values[1] or 1) - 1 if len(values) > 1 else 0
        elif final == "G":
            self.col = value - 1
        elif final == "d":
            self.row = value - 1
        elif final == "A":
            self.row -= value
        elif final == "B":
            self.row += value
        elif final == "C":
            self.col += value
        elif final == "D":
            self.col -= value
        elif final == "J" and values[0] in (2, 3):
            self.cells = [[" "] * 120 for _ in range(40)]
        elif final == "K":
            start = 0 if values[0] in (1, 2) else self.col
            end = self.col + 1 if values[0] == 1 else 120
            self.cells[self.row][start:end] = [" "] * (end - start)
        else:
            raise RuntimeError(f"unsupported terminal CSI {sequence!r}")
        self.row = max(0, min(self.row, 39))
        self.col = max(0, min(self.col, 119))

    def osc(self, sequence: str) -> None:
        payload = sequence[2:].removesuffix("\x07").removesuffix("\x1b\\")
        if payload not in ("10;?", "11;?"):
            return
        color = payload[:2]
        # Reply even to late probes, but never let their timeout mask a failure.
        self.send(f"\x1b]{color};rgb:1111/1111/1111\x1b\\".encode())
        if color == "11":
            self.background_queries += 1
            if self.active:
                raise RuntimeError("OSC 11 background query after TUI startup: terminal input collision")

    def feed(self, data: bytes) -> None:
        self.pending += self.decoder.decode(data)
        self.transcript = (self.transcript + data.decode("utf-8", "replace"))[-6000:]
        while self.pending:
            char = self.pending[0]
            if char == "\x1b":
                if len(self.pending) < 2:
                    return
                prefix = self.pending[1]
                if prefix == "[":
                    match = re.match(r"\x1b\[[0-?]*[ -/]*[@-~]", self.pending)
                    if match is None:
                        return
                    sequence = match.group()
                    self.csi(sequence)
                elif prefix == "]":
                    match = re.search(r"\x07|\x1b\\", self.pending)
                    if match is None:
                        return
                    sequence = self.pending[:match.end()]
                    self.osc(sequence)
                elif prefix in "()":
                    if len(self.pending) < 3:
                        return
                    sequence = self.pending[:3]
                else:
                    sequence = self.pending[:2]
                self.pending = self.pending[len(sequence):]
                continue
            self.pending = self.pending[1:]
            if char == "\r":
                self.col = 0
            elif char == "\n":
                self.row = min(self.row + 1, 39)
            elif char == "\b":
                self.col = max(0, self.col - 1)
            elif char >= " " and not unicodedata.combining(char) and char not in "\ufe0e\ufe0f\u200d":
                self.cells[self.row][min(self.col, 119)] = char
                self.col = min(119, self.col + (2 if unicodedata.east_asian_width(char) in "WF" else 1))
        if not self.manual_slash and "Which ticket would you like to view?" in self.text():
            raise RuntimeError("unsolicited ticket prompt before manual slash")


def _controlling_terminal() -> None:
    os.setsid()
    fcntl.ioctl(0, termios.TIOCSCTTY, 0)


def run(command: list[str], timeout: float) -> None:
    """Run the bounded, frame-driven list/open/navigate/refresh/exit scenario."""
    master, slave = pty.openpty()
    process = None
    terminal = Terminal(master)
    phase = "list"
    deadline = time.monotonic() + timeout
    try:
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
        with tempfile.TemporaryDirectory(prefix="jira-render-pty-") as home:
            environment = {
                "PATH": os.environ.get("PATH", os.defpath),
                "HOME": home, "XDG_CONFIG_HOME": home,
                "TERM": "xterm-256color", "COLORTERM": "truecolor", "LANG": "C.UTF-8",
                "JIRA_CLI_RENDER_PTY_HELPER": "1",
            }
            process = subprocess.Popen(
                command, stdin=slave, stdout=slave, stderr=slave,
                env=environment, preexec_fn=_controlling_terminal,
            )
            os.close(slave)
            slave = -1
            while True:
                remaining = deadline - time.monotonic()
                if remaining <= 0:
                    raise RuntimeError(f"deadline waiting for {phase}; screen={terminal.text()!r}")
                ready, _, _ = select.select([master], [], [], remaining)
                if not ready:
                    continue
                try:
                    data = os.read(master, 65536)
                except OSError as error:
                    if error.errno != errno.EIO:
                        raise
                    data = b""
                if not data:
                    break
                terminal.feed(data)
                screen = terminal.text()
                if phase == "list" and "ROOTLISTREADY" in screen:
                    if not terminal.active or not terminal.background_queries:
                        raise RuntimeError("list frame without real preflight background detection")
                    terminal.send(b"v")
                    phase = "selected issue"
                elif phase == "selected issue" and "SELECTEDTITLE" in screen and "SELECTEDBODY" in screen:
                    terminal.manual_slash = True
                    terminal.send(b"/")
                    phase = "manual prompt"
                elif phase == "manual prompt" and "Which ticket would you like to view?" in screen:
                    terminal.send(b"SECOND-2\r")
                    phase = "navigated issue"
                elif (phase == "navigated issue" and "NAVIGATEDTITLE" in screen and "NAVIGATEDBODY" in screen
                      and "Which ticket would you like to view?" not in screen):
                    terminal.manual_slash = False
                    terminal.send(b"\x12")
                    phase = "refreshed issue"
                elif phase == "refreshed issue" and "REFRESHEDTITLE" in screen and "REFRESHEDBODY" in screen:
                    terminal.send(b"\x03")
                    phase = "clean exit"
            status = process.wait(timeout=max(0.1, deadline - time.monotonic()))
            if status != 0 or phase != "clean exit":
                raise RuntimeError(f"child exit={status}, phase={phase}; fixture tail={terminal.transcript!r}")
            if terminal.pending.startswith("\x1b"):
                raise RuntimeError("child ended with an incomplete terminal escape")
            print(f"PASS: preflight OSC11={terminal.background_queries}, active OSC11=0; v, /SECOND-2, CtrlR, CtrlC")
    finally:
        # Closing the master first also releases Darwin's pending tty drain.
        os.close(master)
        if slave != -1:
            os.close(slave)
        if process is not None and process.poll() is None:
            os.killpg(process.pid, signal.SIGKILL)
            process.wait(timeout=5)


def main() -> int:
    """Parse an explicit helper command; never discover Jira config or a VM."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout", type=float, default=35)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command or args.timeout <= 0:
        parser.error("a helper command and positive timeout are required")
    try:
        run(command, args.timeout)
    except (OSError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(f"FAIL: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
