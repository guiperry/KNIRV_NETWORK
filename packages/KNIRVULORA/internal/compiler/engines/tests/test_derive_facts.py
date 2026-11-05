import unittest

import derive_facts as df

DOC = """---
name: demo
---

# Demo

Check CI on a PR:
```bash
gh pr checks 55 --repo owner/repo
```

- Set `DEMO_SOCKET_DIR` to move sockets.
- Default socket path: `$DEMO_SOCKET_DIR/demo.sock`.

Nothing distinctive here at all, just prose that is long enough to pass a length check.
"""


class DeriveFacts(unittest.TestCase):
    def test_spans_find_flags_envvars_files(self):
        sp = df.spans("Set `DEMO_SOCKET_DIR` and run gh pr checks --repo owner/repo, see demo.sock and ~/.cfg/x.json")
        for want in ("demo_socket_dir", "--repo", "demo.sock", "~/.cfg/x.json"):
            self.assertIn(want, sp)

    def test_chunks_keep_only_passages_with_spans(self):
        ch = df.chunks("demo", df.strip_doc(DOC))
        texts = [c["text"] for c in ch]
        self.assertTrue(any("gh pr checks" in t for t in texts))
        self.assertTrue(any("DEMO_SOCKET_DIR" in t for t in texts))
        self.assertFalse(any("Nothing distinctive" in t for t in texts))

    def test_fenced_block_stays_with_lead_in(self):
        ch = df.chunks("demo", df.strip_doc(DOC))
        c = next(c for c in ch if "gh pr checks" in c["text"])
        self.assertIn("Check CI on a PR", c["text"])

    def test_gate_accepts_grounded_answer_and_rejects_invented_command(self):
        ch = df.chunks("demo", df.strip_doc(DOC))
        c = next(c for c in ch if "gh pr checks" in c["text"])
        corpus = df.strip_doc(DOC).lower()
        self.assertIsNone(df.gate("Run `gh pr checks 55 --repo owner/repo`.", c, corpus))
        self.assertEqual(df.gate("Run `gh pr checks 55 --repo owner/repo --merge-squash`.", c, corpus), "ungrounded")
        self.assertEqual(df.gate("Just look at the CI page.", c, corpus), "recall")
        self.assertEqual(df.gate("   ", c, corpus), "empty")

    def test_tfidf_neighbors_exclude_self_and_prefer_similar(self):
        t = df.TfIdf(["gh pr checks ci status", "tmux socket dir sockets", "pr checks ci runs github"])
        self.assertEqual(t.neighbors(0, 1), [2])
        self.assertNotIn(0, t.neighbors(0, 2))


if __name__ == "__main__":
    unittest.main()
