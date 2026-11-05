"""
Fact sheet for the real-skill spike (Task 5.2a). Every fact is copied from the
named SKILL.md in KNIRVAGENT/workspace/skills; `need` lists the tokens an answer must
contain (case-insensitive) to count as reproducing that fact. Topics A/B are used to
build training questions, topic C only for held-out questions, so held-out prompts
never share wording with training prompts.

skill-creator/SKILL.md (18 KB, ~4.6k tokens) is deliberately excluded: it does not
fit the 4096-token teacher context together with the other skills.
"""

SKILLS_DIR = "/home/gperry/Documents/GitHub/KNIRV/KNIRV_NETWORK/packages/KNIRVAGENT/workspace/skills"

FACTS = {
    "github": [
        (["gh pr checks", "--repo"], ["check the CI status of a pull request", "see whether the checks on PR 55 passed", "see whether a PR's CI is green"]),
        (["gh run list", "--limit"], ["list recent workflow runs", "see the latest GitHub Actions runs", "get the most recent CI runs for a repo"]),
        (["gh run view", "--log-failed"], ["view logs for failed steps only", "read only the failed step logs of a run", "see just what broke in a CI run"]),
        (["--repo owner/repo"], ["use gh when I'm not inside a git directory", "point gh at a repository from outside a checkout", "run gh commands without a local clone"]),
        (["gh api", "--jq"], ["get specific fields of a PR through the API", "query pull request data with gh api and filter it", "pull the title and state of PR 55 via the API"]),
        (["gh issue list", "--json"], ["get issues as structured JSON", "print issue numbers and titles as JSON", "get machine-readable output for issues"]),
    ],
    "tmux": [
        (["NANOBOT_TMUX_SOCKET_DIR"], ["configure the tmux socket directory", "set where the skill keeps its tmux sockets", "change the folder used for tmux sockets"]),
        (["nanobot.sock"], ["find the default tmux socket file name", "know which socket file the skill uses by default", "name the default tmux socket"]),
        (["PYTHON_BASIC_REPL=1"], ["run a Python REPL inside tmux so send-keys works", "start python in a tmux pane without breaking send-keys", "avoid the non-basic REPL problem in tmux"]),
        (["Ctrl+b d"], ["detach from an attached tmux session", "leave an attached tmux session running", "detach without killing the session"]),
        (["wait-for-text.sh", "-p"], ["wait for a prompt to appear in a pane", "block until some text shows up in a tmux pane", "poll a pane for a pattern with a timeout"]),
        (["kill-server"], ["remove everything on the private tmux socket", "tear down every session on my socket at once", "shut down the whole private tmux server"]),
    ],
    "weather": [
        (["wttr.in", "format=3"], ["get a one-line weather report", "get a quick one-liner forecast for a city", "print the weather for London on one line"]),
        (["open-meteo"], ["get weather as JSON when wttr.in is unavailable", "use the free fallback weather API", "get programmatic weather data without an API key"]),
        (["?m"], ["show weather in metric units", "switch wttr.in to metric", "get Celsius output from wttr.in"]),
        (["?1"], ["show only today's forecast", "limit wttr.in to today", "get today-only weather"]),
        (["New+York"], ["query weather for a city with a space in its name", "URL-encode a city name that has a space", "ask wttr.in about New York"]),
        ([".png"], ["get the weather as an image", "download a weather report as a PNG", "save the forecast as a picture"]),
    ],
    "summarize": [
        (["google/gemini-3-flash-preview"], ["know the default summarize model", "find out which model summarize uses if none is set", "name the model summarize defaults to"]),
        (["GEMINI_API_KEY"], ["set the API key for Google models in summarize", "configure the Google provider key", "name the env var for the Google key"]),
        (["--extract-only"], ["get just the transcript of a YouTube link", "extract text from a URL without summarizing", "pull the raw transcript only"]),
        (["~/.summarize/config.json"], ["find the summarize config file", "set a default model in a config file", "know where summarize reads its settings"]),
        (["APIFY_API_TOKEN"], ["enable the YouTube fallback in summarize", "set the token for the Apify YouTube fallback", "get transcripts when the normal extraction fails"]),
        (["FIRECRAWL_API_KEY"], ["summarize a site that blocks extraction", "handle a blocked website in summarize", "configure the extraction fallback for blocked sites"]),
    ],
    "hardware": [
        (["/etc/init.d/S30wifi stop"], ["free the shared I2C pins from WiFi", "stop WiFi before using I2C on a Sipeed board", "release pins that WiFi shares"]),
        (["modprobe i2c-dev"], ["load the I2C device module", "fix 'No I2C buses found'", "enable i2c-dev on the board"]),
        (["confirm: true"], ["do a write operation safely", "perform an I2C write", "know what a write needs before it runs"]),
        (["0x03", "0x77"], ["know the valid I2C address range", "check which 7-bit addresses are accepted", "see the allowed I2C addresses"]),
        (["4096"], ["know the maximum SPI transaction size", "find the SPI per-transfer limit", "check how many bytes one SPI transfer can carry"]),
        (["4.7k"], ["pick the typical I2C pull-up resistor", "choose pull-up values when a scan finds nothing", "check which pull-up resistance is typical"]),
    ],
}

TRAIN_FRAMES = [
    "How do I {t}?",
    "What's the way to {t}?",
    "I need to {t}. What should I run?",
    "Quick question: how can I {t}?",
    "Walk me through how to {t}.",
    "What does the {s} skill say about how to {t}?",
    "Help me {t}.",
    "Tell me how to {t}, with the exact command or setting.",
]
HELD_FRAMES = [
    "Can you explain how I can {t}?",
    "Using the {s} skill, how would I {t}?",
    "Remind me, how do I {t}?",
    "Short answer please: {t}.",
]
