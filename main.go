// penguin-system is a small interactive exam that asks questions partly
// derived from the machine it's actually running on. Read-only, no
// external commands, no dependencies beyond the standard library;>>.
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const examSize = 8

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// ---- terminal ----

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

var useColor = func() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}()

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + ansiReset
}

// ---- the mascot ----
// Three moods. Kept small on purpose — this is a quiz, not an art gallery.

const penguinCalm = `  .--.
 (o o)
 ( : )
  ^^^`

const penguinWorried = `  .--.
 (O O)
 ( : )
  ^^^`

const penguinPanic = `  .--.
 (@ @)
 ( : )
 /|||\
 PANIC`

func printPenguin(art string) {
	fmt.Println(paint(ansiCyan, art))
}

// ---- questions ----

type question struct {
	category string
	text     string
	choices  [4]string
	correct  int // index into choices
	explain  string
	fact     string // shown after the answer, may be empty
}

// newQuestion assembles a question from one correct answer and exactly
// three distractors, shuffling them so the correct one isn't always #1.
func newQuestion(category, text, correctText string, distractors [3]string, explain, fact string) question {
	choices := [4]string{correctText, distractors[0], distractors[1], distractors[2]}
	correctIdx := 0
	rng.Shuffle(4, func(i, j int) {
		if i == correctIdx {
			correctIdx = j
		} else if j == correctIdx {
			correctIdx = i
		}
		choices[i], choices[j] = choices[j], choices[i]
	})
	return question{category: category, text: text, choices: choices, correct: correctIdx, explain: explain, fact: fact}
}

// distractorsInt builds three plausible-but-wrong integers around a real
// value. Never used to invent the *correct* answer, only the wrong ones.
func distractorsInt(correct int) [3]string {
	candidates := []int{correct + 1, correct + 2, correct + 3, correct - 1, correct + 5, correct - 2, correct * 2}
	var out []int
	seen := map[int]bool{correct: true}
	for _, v := range candidates {
		if v > 0 && !seen[v] {
			seen[v] = true
			out = append(out, v)
			if len(out) == 3 {
				break
			}
		}
	}
	for len(out) < 3 { // only reached for pathologically small correct values
		v := correct + len(out) + 10
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return [3]string{strconv.Itoa(out[0]), strconv.Itoa(out[1]), strconv.Itoa(out[2])}
}

func readFileTrim(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// ---- generators: one per category, each reads the real system and
// returns ok=false rather than ever making something up ----

func questionCPU() (question, bool) {
	count := 0
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		count = strings.Count(string(data), "processor\t:")
	}
	if count <= 0 {
		count = runtime.NumCPU() // still a real value, just from a different source
	}
	if count <= 0 {
		return question{}, false
	}
	correct := strconv.Itoa(count)
	text := fmt.Sprintf("This machine currently reports %d logical CPU(s) to the kernel. How many does /proc/cpuinfo list?", count)
	return newQuestion("cpu", text, correct, distractorsInt(count),
		"/proc/cpuinfo has one \"processor\" entry per logical CPU, which includes hyperthreads, not just physical cores.", ""), true
}

func questionMemory() (question, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return question{}, false
	}
	kb := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.Atoi(fields[1]); err == nil {
					kb = v
				}
			}
			break
		}
	}
	if kb <= 0 {
		return question{}, false
	}
	if kb >= 1024*1024 {
		gib := kb / (1024 * 1024)
		text := "According to /proc/meminfo, this machine has roughly how much total RAM?"
		correct := fmt.Sprintf("~%d GiB", gib)
		d := distractorsInt(gib)
		var dd [3]string
		for i, v := range d {
			dd[i] = "~" + v + " GiB"
		}
		return newQuestion("memory", text, correct, dd,
			"MemTotal in /proc/meminfo is reported in kB; divide by 1024 twice to get GiB.", ""), true
	}
	mib := kb / 1024
	if mib <= 0 {
		return question{}, false
	}
	text := "According to /proc/meminfo, this machine has roughly how much total RAM?"
	correct := fmt.Sprintf("~%d MiB", mib)
	d := distractorsInt(mib)
	var dd [3]string
	for i, v := range d {
		dd[i] = "~" + v + " MiB"
	}
	return newQuestion("memory", text, correct, dd,
		"Small MemTotal values like this are typical of containers and VMs with a tight memory cap.", ""), true
}

func questionNetwork() (question, bool) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil || len(entries) == 0 {
		return question{}, false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	count := len(names)
	text := fmt.Sprintf("This system currently exposes %d entries under /sys/class/net (loopback included). How many?", count)
	fact := "interfaces seen: " + strings.Join(names, ", ")
	return newQuestion("network", text, strconv.Itoa(count), distractorsInt(count),
		"Every network interface the kernel knows about gets a directory under /sys/class/net, including \"lo\".", fact), true
}

func questionProcess() (question, bool) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return question{}, false
	}
	threads := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Threads:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if v, err := strconv.Atoi(fields[1]); err == nil {
					threads = v
				}
			}
			break
		}
	}
	if threads <= 0 {
		return question{}, false
	}
	text := fmt.Sprintf("Right now, this exam's own process reports a Threads count of %d in its own /proc/self/status. What does it say?", threads)
	return newQuestion("process", text, strconv.Itoa(threads), distractorsInt(threads),
		"/proc/self always refers to whichever process reads it — every process sees its own /proc/self.", ""), true
}

func questionUptime() (question, bool) {
	data, err := readFileTrim("/proc/uptime")
	if err != nil {
		return question{}, false
	}
	fields := strings.Fields(data)
	if len(fields) < 1 {
		return question{}, false
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || seconds < 0 {
		return question{}, false
	}
	buckets := []string{"under 1 hour", "1 hour to 1 day", "1 day to 1 week", "more than 1 week"}
	var correctIdx int
	switch {
	case seconds < 3600:
		correctIdx = 0
	case seconds < 86400:
		correctIdx = 1
	case seconds < 7*86400:
		correctIdx = 2
	default:
		correctIdx = 3
	}
	var distractors [3]string
	j := 0
	for i, b := range buckets {
		if i == correctIdx {
			continue
		}
		distractors[j] = b
		j++
	}
	text := "Reading the first field of /proc/uptime, this machine has been up for roughly:"
	return newQuestion("uptime", text, buckets[correctIdx], distractors,
		"The first number in /proc/uptime is seconds since boot; the second is total idle time summed across CPUs.", ""), true
}

func questionLoad() (question, bool) {
	data, err := readFileTrim("/proc/loadavg")
	if err != nil {
		return question{}, false
	}
	fields := strings.Fields(data)
	if len(fields) < 1 {
		return question{}, false
	}
	text := fmt.Sprintf("This system's current 1-minute load average is %s. What does that number actually represent?", fields[0])
	correct := "an average count of processes runnable or waiting on I/O"
	distractors := [3]string{
		"current CPU temperature",
		"percentage of CPU used in the last minute",
		"amount of free memory in megabytes",
	}
	return newQuestion("load", text, correct, distractors,
		"Load average is not a percentage — on a 4-core box, a load of 4.0 means the CPUs are fully booked, not \"400% busy\".", ""), true
}

func questionKernel() (question, bool) {
	release, err := readFileTrim("/proc/sys/kernel/osrelease")
	if err != nil || release == "" {
		version, verr := readFileTrim("/proc/version")
		if verr != nil {
			return question{}, false
		}
		fields := strings.Fields(version)
		found := false
		for i, f := range fields {
			if f == "version" && i+1 < len(fields) {
				release = fields[i+1]
				found = true
				break
			}
		}
		if !found || release == "" {
			return question{}, false
		}
	}
	decoys := []string{"5.15.0-91-generic", "4.19.0-24-amd64", "6.1.0-13-amd64", "5.4.0-42-generic"}
	var distractors [3]string
	j := 0
	for _, d := range decoys {
		if d == release {
			continue
		}
		distractors[j] = d
		j++
		if j == 3 {
			break
		}
	}
	if j < 3 {
		return question{}, false
	}
	text := "Which of these is the kernel release this system is actually running right now?"
	return newQuestion("kernel", text, release, distractors,
		"uname -r and /proc/sys/kernel/osrelease report the same string — the latter needs no process spawn to read.", ""), true
}

func questionFilesystem() (question, bool) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return question{}, false
	}
	fstype := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "/" {
			fstype = fields[2]
			break
		}
	}
	if fstype == "" {
		return question{}, false
	}
	pool := []string{"ext4", "xfs", "btrfs", "overlay", "zfs", "tmpfs"}
	var distractors [3]string
	j := 0
	for _, f := range pool {
		if f == fstype {
			continue
		}
		distractors[j] = f
		j++
		if j == 3 {
			break
		}
	}
	if j < 3 {
		return question{}, false
	}
	text := "According to /proc/mounts, what filesystem type is your root (/) currently mounted as?"
	return newQuestion("filesystem", text, fstype, distractors,
		"Containers often show \"overlay\" here since the root is a union of image layers, not a plain disk filesystem.", ""), true
}

// static fallback questions, used only if the machine can't supply enough
// real ones (heavily locked down /proc, weird container, etc).

func staticQuestions() []question {
	return []question{
		newQuestion("concepts",
			"In Unix permission notation, what does chmod 644 grant?",
			"owner: read/write, group: read, others: read",
			[3]string{
				"owner: read/write/execute, everyone else: nothing",
				"everyone: read/write",
				"owner: execute only, everyone else: read",
			},
			"6 = rw-, 4 = r--, 4 = r--, read left to right as owner/group/other.", ""),
		newQuestion("concepts",
			"What does SIGKILL (signal 9) let a process do before it dies?",
			"nothing — it cannot be caught, blocked, or ignored",
			[3]string{
				"run any registered cleanup handlers first",
				"flush its open file buffers, then exit",
				"choose to ignore it if it's in a critical section",
			},
			"SIGTERM is the polite one processes can catch; SIGKILL is enforced by the kernel directly.", ""),
		newQuestion("concepts",
			"What is /proc, structurally speaking?",
			"a virtual filesystem generated by the kernel, not stored on disk",
			[3]string{
				"a hidden directory the installer writes at setup time",
				"a compressed log archive rotated by systemd",
				"a regular directory on the root filesystem's disk",
			},
			"procfs is generated on the fly; reading a file there runs kernel code, not a disk read.", ""),
		newQuestion("concepts",
			"An inode stores metadata about a file. Which of these does it NOT store?",
			"the file's name",
			[3]string{"the file's owner", "the file's permission bits", "pointers to the file's data blocks"},
			"Filenames live in directory entries, which just map a name to an inode number.", ""),
	}
}

func buildExam() []question {
	generators := []func() (question, bool){
		questionCPU, questionMemory, questionNetwork, questionProcess,
		questionUptime, questionLoad, questionKernel, questionFilesystem,
	}
	var exam []question
	for _, gen := range generators {
		if q, ok := gen(); ok {
			exam = append(exam, q)
		}
	}
	if len(exam) < examSize {
		fallback := staticQuestions()
		rng.Shuffle(len(fallback), func(i, j int) { fallback[i], fallback[j] = fallback[j], fallback[i] })
		for _, q := range fallback {
			if len(exam) >= examSize {
				break
			}
			exam = append(exam, q)
		}
	}
	rng.Shuffle(len(exam), func(i, j int) { exam[i], exam[j] = exam[j], exam[i] })
	if len(exam) > examSize {
		exam = exam[:examSize]
	}
	return exam
}

// ---- exam loop ----

// readAnswer returns a 1-4 choice, or ok=false on EOF so the caller can
// end the exam early instead of looping forever.
func readAnswer(scanner *bufio.Scanner) (int, bool) {
	for {
		fmt.Print(paint(ansiDim, "answer [1-4]> "))
		if !scanner.Scan() {
			return 0, false
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Println("Pick 1-4.")
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > 4 {
			fmt.Println("Pick 1-4.")
			continue
		}
		return n - 1, true
	}
}

func rank(percent int) string {
	switch {
	case percent < 30:
		return "terminal tourist"
	case percent < 50:
		return "sudo apprentice"
	case percent < 70:
		return "package updater"
	case percent < 85:
		return "sudo enjoyer"
	case percent < 95:
		return "shell resident"
	default:
		return "kernel acquaintance"
	}
}

func main() {
	fmt.Println(paint(ansiBold, "penguin-system"))
	fmt.Println(paint(ansiDim, "a small Linux exam for people who claim they know Linux"))
	fmt.Println()
	printPenguin(penguinCalm)
	fmt.Println()

	exam := buildExam()
	if len(exam) == 0 {
		fmt.Println("Couldn't read enough of this system (or anything at all) to build an exam.")
		fmt.Println("That's unusual. Check that /proc is mounted.")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	score := 0
	correctCount := 0
	wrongByCategory := map[string]int{}
	askedByCategory := map[string]int{}
	answeredCount := 0

	for i, q := range exam {
		fmt.Printf("%s\n", paint(ansiCyan, fmt.Sprintf("[%d/%d] %s", i+1, len(exam), strings.ToUpper(q.category))))
		fmt.Println(q.text)
		for j, choice := range q.choices {
			fmt.Printf("  %d) %s\n", j+1, choice)
		}
		askedByCategory[q.category]++

		idx, ok := readAnswer(scanner)
		if !ok {
			fmt.Println()
			fmt.Println("(EOF — ending the exam early.)")
			break
		}
		answeredCount++

		if idx == q.correct {
			score += 10
			correctCount++
			fmt.Println(paint(ansiGreen, "✓ Correct."))
			if score > 30 {
				fmt.Println("Penguin status: slightly less worried.")
			}
		} else {
			wrongByCategory[q.category]++
			fmt.Println(paint(ansiRed, "✗ Nope.") + " " + q.explain)
			fmt.Printf("  (correct answer: %s)\n", q.choices[q.correct])
			if wrongByCategory[q.category] >= 2 {
				printPenguin(penguinPanic)
			} else {
				printPenguin(penguinWorried)
			}
		}
		if q.fact != "" {
			fmt.Println(paint(ansiDim, "  "+q.fact))
		}
		fmt.Println()
	}

	total := len(exam)
	maxScore := total * 10
	percent := 0
	if maxScore > 0 {
		percent = score * 100 / maxScore
	}

	var weak []string
	for cat, wrong := range wrongByCategory {
		if wrong >= askedByCategory[cat] && askedByCategory[cat] > 0 {
			weak = append(weak, cat)
		}
	}

	fmt.Println(paint(ansiBold, "╭──────── RESULTS ────────╮"))
	fmt.Printf("Score       %d/%d\n", score, maxScore)
	fmt.Printf("Correct     %d/%d\n", correctCount, total)
	if answeredCount < total {
		fmt.Printf("(only %d/%d questions were answered)\n", answeredCount, total)
	}
	fmt.Println()
	fmt.Println("Rank:")
	fmt.Println(" ", rank(percent))
	if len(weak) > 0 {
		fmt.Println()
		fmt.Println("Weak spots:")
		for _, w := range weak {
			fmt.Println(" ", w)
		}
	}
	fmt.Println()
	fmt.Println("Penguin status:")
	if percent >= 70 {
		printPenguin(penguinCalm)
		fmt.Println("Unbothered.")
	} else if percent >= 40 {
		printPenguin(penguinWorried)
		fmt.Println("Still alive.")
	} else {
		printPenguin(penguinPanic)
		fmt.Println("Considering a career change.")
	}
	fmt.Println(paint(ansiBold, "╰──────────────────────────╯"))
}
