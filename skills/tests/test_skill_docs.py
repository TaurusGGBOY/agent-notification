import unittest
from pathlib import Path


SKILL_MD = Path(__file__).resolve().parents[1] / "SKILL.md"


class SkillDocsTests(unittest.TestCase):
    def test_skill_defaults_to_running_setup_script(self):
        text = SKILL_MD.read_text(encoding="utf-8")

        self.assertIn("## Default Workflow", text)
        self.assertIn("scripts/setup.py", text)
        self.assertIn("--agents claude codex openclaw", text)
        self.assertIn("configure_openclaw.py", text)
        self.assertIn("Do not only explain", text)
        self.assertIn("standard library", text)
        self.assertNotIn("zeroconf", text.lower())


if __name__ == "__main__":
    unittest.main()
