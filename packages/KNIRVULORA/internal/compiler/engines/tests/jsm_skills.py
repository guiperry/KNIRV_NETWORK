"""
Fact sheet for the second real-skill set: jsmastery-pro/skills (MIT, commit 43b69e4),
four workflow skills: debug, check, document, scope. Facts are copied from each
SKILL.md; `need` tokens are matched case-insensitively. Topics A/B train, C held-out.
The repo is cloned into a scratch directory and only READ; nothing in it is executed.
architect/audit/develop/sync/test (13-28 KB each) are excluded: with the four above they
exceed the teacher context.
"""
import os
from real_skills import TRAIN_FRAMES, HELD_FRAMES  # noqa: F401

SKILLS_DIR = os.environ.get("JSM_SKILLS_DIR", "")
DOC_PATH = "{dir}/{name}/SKILL.md"
STRIP_OUTPUT_STYLE = True

FACTS = {
    "debug": [
        (["reproduce", "localize", "hypothes", "verify"], ["know the loop that /debug runs", "see the steps /debug follows to find a root cause", "explain the investigation loop in the debug skill"]),
        (["at a time"], ["decide how many hypotheses to test per experiment", "know how many theories to try in one round", "avoid changing several things at once while debugging"]),
        (["/test", "regression"], ["stop a fixed bug from coming back", "lock in a fix after /debug", "protect a bugfix with a test"]),
        (["/architect"], ["handle a bug caused by a flawed design decision", "route a design flaw found while debugging", "treat a bug that is really a wrong decision"]),
        (["/loop"], ["tell /debug's loop apart from another skill that repeats a command", "poll a flaky test over time", "watch something across many runs instead of one investigation"]),
        (["sonnet"], ["pick the model for a debug subagent in Claude Code", "set the model for the investigation subagent", "choose which model a debug subagent should use"]),
    ],
    "check": [
        (["verify", "review"], ["list the modes of /check", "know what /check can do", "see which two jobs /check has"]),
        (["docs/reviews"], ["find where /check review writes findings", "locate the review output", "know the folder for code review findings"]),
        (["blind spot"], ["understand why review uses a different model", "explain why the reviewer should not be the author's model", "justify a fresh model for code review"]),
        (["/debug"], ["handle a failure found by /check verify", "know where verify failures are sent", "route a verify failure"]),
        (["both"], ["respond when /check is run without a mode", "handle a bare /check", "know what happens if no mode word is given"]),
        (["/develop"], ["know when to run /check verify", "time the verify mode", "place /check verify in the workflow"]),
    ],
    "document": [
        (["changelog", "release-note", "postmortem"], ["list the document types /document writes", "know the four kinds of documents", "see what /document can produce"]),
        (["CHANGELOG.md", "Keep a Changelog"], ["find where changelog entries go and in what format", "know the changelog file and its style", "see the changelog convention"]),
        (["docs/postmortems"], ["find where postmortems are saved", "locate postmortem output", "know the postmortem folder"]),
        (["docs/releases"], ["find where release notes are written", "locate release note files", "know the release notes folder"]),
        (["gh pr edit"], ["update a PR that already exists", "change the body of an existing pull request", "handle a PR that was already opened"]),
        (["haiku"], ["pick the model for the read only scout subagent", "choose the cheapest model for reading a large diff", "set the scout subagent model"]),
    ],
    "scope": [
        (["docs/scope"], ["find where the scope is stored", "locate the living scope files", "know the scope folder"]),
        (["prototype", "alpha", "beta"], ["list the workflow tiers", "know the rigor levels for a feature", "see the tier ladder in /scope"]),
        (["planned"], ["know the starting status of a new feature", "see what status new features get", "find the initial feature status"]),
        (["dropped"], ["handle a feature removed from scope", "mark a feature that is no longer planned", "keep history for removed features"]),
        (["/architect"], ["know who picks tools and libraries", "find out where tool choices are made", "see which skill chooses the stack"]),
        (["index.md"], ["split a large product's scope into epics", "organise the scope files for a big product", "know the file name for the epic index"]),
    ],
}
