"""Unit tests for span_verifier (run: python -m unittest test_span_verifier, from this directory)."""
import unittest

import span_verifier as sv


class Normalise(unittest.TestCase):
    def test_case_and_whitespace(self):
        self.assertEqual(sv.normalise("  Run\n\n`Git  Status`\t"), "run 'git status'")

    def test_curly_quotes_and_nfkc(self):
        self.assertEqual(sv.normalise("“Foo” Ａ"), '"foo" a')

    def test_span_edges_and_trailing_punct(self):
        self.assertEqual(sv.normalise_span("`git status`."), "git status")
        self.assertEqual(sv.normalise_span("(--force),"), "--force")
        self.assertEqual(sv.normalise_span("~/.config/app/"), "~/.config/app/")


class Present(unittest.TestCase):
    P = "Use `docker ps -a` to list containers. Set NRV_LOG_LEVEL=debug in .env. Tools live in /etc/knirv/tools."

    def test_exact_and_case_insensitive(self):
        self.assertTrue(sv.present("docker ps -a", self.P))
        self.assertTrue(sv.present("DOCKER PS -A", self.P))

    def test_token_boundary(self):
        self.assertFalse(sv.present("ls", "the tools folder"))
        self.assertTrue(sv.present("ls", "run ls now"))

    def test_path_and_env_present(self):
        self.assertTrue(sv.present("/etc/knirv/tools", self.P))
        self.assertTrue(sv.present("NRV_LOG_LEVEL", self.P))

    def test_absent(self):
        self.assertFalse(sv.present("docker rm", self.P))
        self.assertFalse(sv.present("", self.P))

    def test_punctuation_led_span_has_no_left_boundary_requirement(self):
        self.assertTrue(sv.present("--force", "use git push --force carefully"))
        self.assertFalse(sv.present("--forced", "use git push --force carefully"))


class JoinerBoundaries(unittest.TestCase):
    """I_52 tightening (owner answer 23a): . - / join a span to adjacent token text."""

    def test_version_prefix_not_inside_longer_version(self):
        self.assertFalse(sv.present("v1.2", "release v1.2.3 is out"))
        self.assertTrue(sv.present("v1.2.3", "release v1.2.3 is out"))

    def test_package_name_not_inside_hyphenated_name(self):
        self.assertFalse(sv.present("pytest", "install pytest-cov for coverage"))
        self.assertTrue(sv.present("pytest-cov", "install pytest-cov for coverage"))

    def test_path_not_inside_longer_path(self):
        self.assertFalse(sv.present("/etc/x", "config under /etc/x/y"))
        self.assertTrue(sv.present("/etc/x/y", "config under /etc/x/y"))

    def test_joiner_on_left_joins_too(self):
        self.assertFalse(sv.present("pytest", "old-pytest tooling"))
        self.assertFalse(sv.present("/etc/x", "foo/etc/x"))
        self.assertFalse(sv.present("pytest", "use pytest.py here"))

    def test_trailing_sentence_period_does_not_join(self):
        self.assertTrue(sv.present("pytest", "uses pytest."))
        self.assertTrue(sv.present("pytest", "(pytest)"))
        self.assertTrue(sv.present("pytest", "pytest..."))

    def test_non_ascii_letters_are_word_characters(self):
        self.assertFalse(sv.present("élan", "the xélan engine"))
        self.assertFalse(sv.present("élan", "the élanx engine"))
        self.assertTrue(sv.present("élan", "the élan engine"))
        self.assertFalse(sv.present("ls", "élsé"))

    def test_combining_mark_is_word_character(self):
        # Devanagari vowel sign (Mc) after क: कि is one token, so क is not a boundary match
        self.assertFalse(sv.present("क", "यह कि है"))
        self.assertTrue(sv.present("क", "यह क है"))
        # combining diaeresis (Mn) after s: ls̈x is one token
        self.assertFalse(sv.present("ls", "ls̈x"))
        self.assertTrue(sv.present("ls", "run ls now"))

    def test_format_character_joins_token(self):
        # zero-width space (Cf) is not a boundary: élan<ZWSP>x is one token
        self.assertFalse(sv.present("élan", "élan​x"))
        self.assertFalse(sv.present("élan", "x​élan"))
        # soft hyphen (Cf) inside the passage token is not a boundary either
        self.assertFalse(sv.present("pyt", "pyt­hon"))

    def test_digit_leading_span(self):
        self.assertTrue(sv.present("2fa", "enable 2fa today"))
        self.assertFalse(sv.present("2fa", "x2fa today"))
        self.assertFalse(sv.present("2fa", "12fa today"))

    def test_span_starting_or_ending_with_joiner(self):
        self.assertTrue(sv.present(".env", "set .env now"))
        self.assertFalse(sv.present(".env", "set x.env now"))
        self.assertTrue(sv.present("pytest-", "pytest- is a prefix"))
        self.assertFalse(sv.present("pytest-", "pytest-cov"))
        self.assertTrue(sv.present("tests/", "under tests/ only"))
        self.assertFalse(sv.present("tests/", "under tests/x only"))

    def test_overlapping_occurrence_found(self):
        # "aba" starts at 0 (joined to "ba" on the right) and at 2 (valid); the valid one must count
        self.assertTrue(sv.present("aba", "ababa aba"))


class Supported(unittest.TestCase):
    P = "Run `pytest -q tests/` from the repo root. Config is in pyproject.toml."

    def test_all_spans_in_passage(self):
        self.assertTrue(sv.supported("Run `pytest -q tests/` to test.", self.P))

    def test_invented_span_refused(self):
        a = "Run `pytest --fast` from the repo root."
        self.assertFalse(sv.supported(a, self.P))
        self.assertEqual(sv.unsupported_spans(a, self.P), ["pytest --fast", "--fast"])

    def test_prose_without_spans_is_vacuously_supported(self):
        self.assertTrue(sv.supported("Look at the repo root and run the tests.", self.P))

    def test_empty_answer_not_supported(self):
        self.assertFalse(sv.supported("   ", self.P))
        self.assertFalse(sv.supported("", self.P))

    def test_duplicate_spans_counted_once(self):
        a = "Run `nope/x` then `nope/x` again."
        self.assertEqual(sv.unsupported_spans(a, self.P), ["nope/x"])


class BestPassage(unittest.TestCase):
    def test_picks_most_similar(self):
        import derive_facts as df
        texts = ["rotate logs with logrotate config", "start the docker daemon service",
                 "check disk usage with du command"]
        tf = df.TfIdf(texts)
        self.assertEqual(sv.best_passage_index(tf, "how do I check disk usage?"), 2)


if __name__ == "__main__":
    unittest.main()
