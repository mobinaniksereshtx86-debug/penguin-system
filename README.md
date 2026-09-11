# penguin-system

A small Linux exam for people who claim they know Linux.

It asks about 8 questions, mostly pulled from the machine you're actually
running it on — your real CPU count, your real memory total, your real
uptime, and also your real kernel version:>. Get one wrong and the mascot gets more
worried. That's it. That's the whole feature set...

```
penguin-system
a small Linux exam for people who claim they know Linux

  .--.
 (o o)
 ( : )
  ^^^

[1/8] CPU
This machine currently reports 8 logical CPU(s) to the kernel. How many
does /proc/cpuinfo list?
  1) 6
  2) 9
  3) 8
  4) 10
answer [1-4]> 3
✓ Correct.

[2/8] KERNEL
Which of these is the kernel release this system is actually running
right now?
  1) 5.15.0-91-generic
  2) 6.8.0-45-generic
  3) 4.19.0-24-amd64
  4) 5.4.0-42-generic
answer [1-4]> 1
✗ Nope. uname -r and /proc/sys/kernel/osrelease report the same string —
the latter needs no process spawn to read.
  (correct answer: 6.8.0-45-generic)

  .--.
 (O O)
 ( : )
  ^^^
```

## Why

Most "Linux quiz" tools ask the same textbook questions everywhere. This
one reads `/proc` and `/sys` on your actual machine and asks about that
instead, so the answer key is your own computer, not a memorized fact.
If you genuinely understand what `/proc/loadavg` or `/proc/self/status`
is telling you, this is easy. If you've only ever pasted commands from
search results, it isn't.

## Build

```
go build
```

That's it. No dependencies, no Makefile, no config. One `go.mod`, one
`main.go`.

```
./penguin-system
```

## What it reads

All read-only, all standard `/proc` and `/sys` pseudo-files:

- `/proc/cpuinfo` — logical CPU count
- `/proc/meminfo` — total memory
- `/sys/class/net` — network interfaces
- `/proc/self/status` — this process's own thread count
- `/proc/uptime` — system uptime
- `/proc/loadavg` — load average
- `/proc/sys/kernel/osrelease` (falls back to `/proc/version`) — kernel release
- `/proc/mounts` — root filesystem type

It never writes anywhere, never runs another program, and never touches
the network. If a value can't be read, or looks malformed, that question
is silently skipped and a small set of static fallback questions fills
the gap instead. Nothing is ever guessed and presented as a real system
fact.

## Supported platform

Linux. It depends on `/proc` and `/sys` existing, so it won't do
anything useful on macOS, BSD, or Windows — it'll just run out of
questions and fall back to the static pool. Works fine over SSH, in a
container, in a VM, on anything from a Raspberry Pi to a build server.

## Design notes

- No third-party packages, no TUI framework, no `os/exec`. Standard
  library only, on purpose — a Linux quiz that shells out to run
  commands would be missing the point.
- Terminal output is plain and mostly readable without color. Color
  is skipped automatically if `NO_COLOR` is set or `TERM=dumb`.
- Invalid input (empty lines, letters, out-of-range numbers, Ctrl+D)
  never crashes it and never loops forever.
- The penguin is an original ASCII doodle, not Tux.

## License

MIT. See `LICENSE`.
