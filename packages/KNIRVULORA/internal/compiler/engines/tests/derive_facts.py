"""
E5 (derived fact gates) and E2 (nearest-neighbour context) for the skill->ULoRA corpus.

Nothing here calls a model. A skill document is split into passages; each passage's
"facts" are its distinctive literal spans (inline code, flags, paths, ENV_VARS, file
names). Those spans are the ground truth:

  recall    an answer reproduces enough of the spans of the passage it was asked about
  grounded  every distinctive span the answer WRITES also exists somewhere in the
            cluster's documents (a span the documents never contain is an invention)

A record is accepted only if both hold. The same two checks score held-out answers, so
"invented command" becomes a number instead of an anecdote.
"""
import math
import re
from collections import Counter

INLINE = re.compile(r"`([^`\n]{2,100})`")
FLAG = re.compile(r"(?<![\w-])--?[a-zA-Z][\w-]{1,}")
ENVVAR = re.compile(r"\b[A-Z][A-Z0-9]*_[A-Z0-9_]{2,}\b")
PATHLIKE = re.compile(r"(?<![\w])(?:~|\.{0,2})?/?[\w.-]+(?:/[\w.<>{}*-]+)+/?")
FILENAME = re.compile(r"\b[\w-]+\.(?:md|sh|json|yaml|yml|toml|py|js|mjs|txt|sock|key|log|conf|cfg)\b")
GENERIC = {"true", "false", "null", "bash", "json", "yes", "no", "and", "or", "the", "a", "ok", "-", "--"}
MIN_SPAN = 3


def strip_doc(txt, strip_output_style=True):
    lines = [l for l in txt.splitlines() if not l.startswith("metadata:")]
    txt = "\n".join(lines)
    if strip_output_style:
        txt = re.sub(r"## Output style.*?<!-- OUTPUT-STYLE:END -->\n?", "", txt, flags=re.S)
    return txt.strip()


def spans(text):
    """Distinctive literal spans in `text`, lower-cased, de-duplicated, order kept."""
    found = []
    for m in INLINE.finditer(text):
        inner = m.group(1).strip()
        if len(inner) >= MIN_SPAN:
            found.append(inner)
            found.extend(FLAG.findall(inner))
    found.extend(FLAG.findall(text))
    found.extend(ENVVAR.findall(text))
    found.extend(FILENAME.findall(text))
    for m in PATHLIKE.findall(text):
        if "/" in m and len(m) >= MIN_SPAN:
            found.append(m)
    out, seen = [], set()
    for s in found:
        k = s.strip().lower().rstrip(".,;:)")
        if len(k) < MIN_SPAN or k in GENERIC or k in seen:
            continue
        seen.add(k)
        out.append(k)
    return out


def chunks(skill, text, min_chars=40, max_chars=1400):
    """Passages: a paragraph or bullet, plus the fenced block that follows its lead-in line."""
    lines = text.splitlines()
    heading, blocks, cur, in_fence = "", [], [], False

    def flush():
        nonlocal cur
        body = "\n".join(cur).strip()
        if body:
            blocks.append((heading, body))
        cur = []

    for ln in lines:
        if ln.strip().startswith("```"):
            in_fence = not in_fence
            cur.append(ln)
            continue
        if in_fence:
            cur.append(ln)
            continue
        if ln.startswith("#"):
            flush()
            heading = ln.lstrip("# ").strip()
            continue
        if not ln.strip():
            flush()
            continue
        if re.match(r"^\s*([-*]|\d+\.)\s", ln) and cur:
            flush()  # each bullet is its own passage
        cur.append(ln)
    flush()
    out = []
    for i, (h, body) in enumerate(blocks):
        body = body[:max_chars]
        sp = spans(body)
        if len(body) >= min_chars and sp:
            out.append({"id": f"{skill}#{i}", "skill": skill, "heading": h, "text": body, "spans": sp})
    return out


def recall_ok(answer, passage_spans, need=2):
    """Answer reproduces at least `need` of the passage's spans (capped at how many it has)."""
    a = answer.lower()
    return sum(1 for s in passage_spans if s in a) >= min(need, len(passage_spans))


def ungrounded(answer, corpus_text_lower):
    """Spans the answer writes that the cluster's documents never contain."""
    bad = []
    for s in spans(answer):
        if s not in corpus_text_lower:
            bad.append(s)
    return bad


def gate(answer, passage, corpus_text_lower, need=2):
    if not answer or not answer.strip():
        return "empty"
    if not recall_ok(answer, passage["spans"], need):
        return "recall"
    if ungrounded(answer, corpus_text_lower):
        return "ungrounded"
    return None


class TfIdf:
    """Tiny TF-IDF over passages, enough for nearest-neighbour context selection."""

    def __init__(self, texts):
        toks = [re.findall(r"[a-z0-9_]{2,}", t.lower()) for t in texts]
        df = Counter(w for ts in toks for w in set(ts))
        n = len(texts)
        self.idf = {w: math.log((1 + n) / (1 + c)) + 1 for w, c in df.items()}
        self.vecs = [self._vec(ts) for ts in toks]

    def _vec(self, ts):
        tf = Counter(ts)
        v = {w: c * self.idf.get(w, 1.0) for w, c in tf.items()}
        norm = math.sqrt(sum(x * x for x in v.values())) or 1.0
        return {w: x / norm for w, x in v.items()}

    def neighbors(self, i, k):
        a = self.vecs[i]
        scored = []
        for j, b in enumerate(self.vecs):
            if j == i:
                continue
            scored.append((sum(x * b.get(w, 0.0) for w, x in a.items()), j))
        scored.sort(key=lambda t: (-t[0], t[1]))
        return [j for _, j in scored[:k]]
