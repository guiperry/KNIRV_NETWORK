"""
Span verifier for retrieval-grounded abstention (I_52 seeds phase 2, owner decision 1).

It replaces the TF-IDF cosine "support" gate. An answer is SUPPORTED only if every literal
span the answer writes (the same extraction as derive_facts.spans) appears in the retrieved
passage. The retrieved passage is the single best TF-IDF match among ALL passages in the corpus
(87 in this spike, across every skill; not restricted to the question's skill or cluster) for
the question. Retrieval is still used to pick WHERE to check; the decision is by spans.

Normalisation (applied to both the span and the passage before comparison):
  1. Unicode NFKC, then casefold.
  2. Curly quotes and backticks become ASCII quotes; each run of whitespace (incl. newlines)
     becomes one space.
  3. Surrounding quotes, brackets and backticks are stripped, then trailing punctuation
     `.,;:!?)` is stripped (repeatedly).
A span is "present" in the passage when its normalised form occurs in the normalised passage
on token boundaries (I_52 span-verifier tightening, owner answer 23a).

Token characters ("word characters") are any Unicode letter or digit (str.isalnum(), so
non-ASCII letters count), `_`, any Unicode combining mark (category M*, e.g. the vowel
signs of Devanagari or a combining diaeresis), and any format character (category Cf, e.g.
zero-width space or soft hyphen). Marks and format characters are part of the token they
attach to, so they are never a boundary: `क` does not match inside `कि`, `ls` does not match
inside `ls̈x`, and `élan` does not match inside `élan<U+200B>x`. The joiners `.`, `-` and `/`
are word-like too, but only when they join the span to more token text:

  - Right side (always checked): the character right after the span must not be a word
    character. It may be a joiner only if the character after that joiner is NOT a word
    character. So `pytest` does not match inside `pytest-cov`, `pytest.py` or `pytest_x`,
    and `v1.2` does not match inside `v1.2.3`.
  - Left side (checked when the span's first character is a word character or a joiner):
    the character right before the span must not be a word character, and a joiner there
    counts only when the character before that joiner is not a word character. So
    `/etc/x` does not match inside `/etc/x/y` (right side) nor `foo/etc/x` (left side).
  - A trailing sentence period is therefore not joined: `uses pytest.` and `(pytest)` both
    match `pytest`, because `.` is followed by the end of text or a non-word character.
  - Spans that themselves start or end with `.`, `-` or `/` are compared by the same rules,
    e.g. `.env` matches in `set .env now` but not in `x.env`.

Spans whose first character is neither a word character nor a joiner (for example `~/`,
`(`, `$`) get no left-side check, as before. Case-insensitive matching is unchanged: both
sides are casefolded in normalise(). Consequences of the Unicode rule: `ls` does NOT match
inside `élsé` (the `é` is a letter, so it is a word character), `élan` does not match inside
`xélan` or `élanx`. Spans shorter than derive_facts.MIN_SPAN never reach this module.

An answer with NO spans is vacuously supported (no span can be invented); the empty answer is
not supported. Verdict is pure string logic: no model, no network.
"""
import re
import unicodedata

from derive_facts import spans as extract_spans

_QUOTES = {"‘": "'", "’": "'", "“": '"', "”": '"', "`": "'"}
_EDGE = "\"'`[](){}<>"
_TRAIL = ".,;:!?)"


def normalise(text):
    t = unicodedata.normalize("NFKC", text).casefold()
    for k, v in _QUOTES.items():
        t = t.replace(k, v)
    t = re.sub(r"\s+", " ", t).strip()
    return t


def normalise_span(span):
    s = normalise(span).strip(_EDGE + " ")
    prev = None
    while prev != s:
        prev = s
        s = s.rstrip(_TRAIL).rstrip()
        s = s.strip(_EDGE + " ")
    return s


_JOIN = ".-/"


def _is_word(ch):
    """Token character: Unicode letter or digit (str.isalnum), underscore, combining mark (M*)
    or format character (Cf). Marks and format characters join the token they attach to."""
    if ch.isalnum() or ch == "_":
        return True
    return unicodedata.category(ch)[0] == "M" or unicodedata.category(ch) == "Cf"


def _left_ok(p, i):
    """Boundary before position i of p (see module docstring: joiners count only when joined)."""
    if i == 0:
        return True
    c = p[i - 1]
    if _is_word(c):
        return False
    if c in _JOIN and i >= 2 and _is_word(p[i - 2]):
        return False
    return True


def _right_ok(p, j):
    """Boundary at position j (just after the span) in p (see module docstring)."""
    if j >= len(p):
        return True
    c = p[j]
    if _is_word(c):
        return False
    if c in _JOIN and j + 1 < len(p) and _is_word(p[j + 1]):
        return False
    return True


def present(span, passage):
    """True if the normalised span occurs in the normalised passage on token boundaries.

    Boundary rule (module docstring): word characters are Unicode letters/digits, `_`,
    combining marks and format characters;
    `.`, `-`, `/` are boundaries only when followed (right) or preceded (left) by a word
    character. The left check runs when the span starts with a word character or a joiner.
    """
    s = normalise_span(span)
    if not s:
        return False
    p = normalise(passage)
    check_left = _is_word(s[0]) or s[0] in _JOIN
    # lookahead-only pattern so every start position is tried, including overlapping ones
    for m in re.finditer("(?=" + re.escape(s) + ")", p):
        i = m.start()
        if (not check_left or _left_ok(p, i)) and _right_ok(p, i + len(s)):
            return True
    return False


def unsupported_spans(answer, passage):
    """Spans the answer writes that the passage does not contain (normalised, de-duplicated)."""
    bad, seen = [], set()
    for s in extract_spans(answer):
        n = normalise_span(s)
        if n in seen:
            continue
        seen.add(n)
        if not present(s, passage):
            bad.append(s)
    return bad


def supported(answer, passage):
    """Owner rule: supported only if all of the answer's spans appear in the retrieved passage."""
    if not answer or not answer.strip():
        return False
    return not unsupported_spans(answer, passage)


def unsupported_spans_topk(answer, passages):
    """Spans the answer writes that NO passage in `passages` contains (top-k mode, I_52 19a).

    Each span is grounded if it is present in any of the given passages, with the same
    span-boundary rule as present(). With one passage this equals unsupported_spans().
    """
    bad, seen = [], set()
    for s in extract_spans(answer):
        n = normalise_span(s)
        if n in seen:
            continue
        seen.add(n)
        if not any(present(s, p) for p in passages):
            bad.append(s)
    return bad


def supported_topk(answer, passages):
    """Top-k gate: supported only if every span the answer writes appears in at least one of
    the k retrieved passages. Empty-answer and no-span rules are the same as supported()."""
    if not answer or not answer.strip():
        return False
    return not unsupported_spans_topk(answer, passages)


def best_passage_indices(tfidf, query_text, k=1):
    """Indices of the k passages with the highest TF-IDF cosine, best first.

    Ties go to the lower index, so k=1 returns exactly [best_passage_index(...)]."""
    import re as _re
    q = tfidf._vec(_re.findall(r"[a-z0-9_]{2,}", query_text.lower()))
    scored = []
    for i, v in enumerate(tfidf.vecs):
        s = sum(x * v.get(w, 0.0) for w, x in q.items())
        scored.append((-s, i))
    scored.sort()
    return [i for _, i in scored[:k]]


def best_passage_index(tfidf, query_text):
    """Index of the passage with the highest TF-IDF cosine to the question (retrieval step)."""
    return best_passage_indices(tfidf, query_text, 1)[0]
