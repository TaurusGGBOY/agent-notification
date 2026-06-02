import json
import os
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parents[1] / "scripts"
sys.path.insert(0, str(SCRIPT_DIR))

import windows_screenshot  # noqa: E402


class FakeImage:
    size = (5120, 1440)

    def save(self, path):
        with open(path, "wb") as f:
            f.write(b"png")


class FakeImageGrab:
    all_screens = None

    @classmethod
    def grab(cls, all_screens=True):
        cls.all_screens = all_screens
        return FakeImage()


class FailingImageGrab:
    @staticmethod
    def grab(all_screens=True):
        raise IOError("screen grab failed")


class WindowsScreenshotTests(unittest.TestCase):
    def test_capture_defaults_to_all_screens_and_writes_status(self):
        with tempfile.TemporaryDirectory() as tmp:
            output = os.path.join(tmp, "screen.png")
            status = os.path.join(tmp, "screen.json")

            code, data = windows_screenshot.capture_screenshot(
                output,
                status_path=status,
                image_grab=FakeImageGrab,
            )

            self.assertEqual(code, 0)
            self.assertTrue(FakeImageGrab.all_screens)
            self.assertEqual(data["size"], [5120, 1440])
            self.assertEqual(data["bytes"], 3)
            self.assertTrue(os.path.exists(output))
            written = json.loads(Path(status).read_text())
            self.assertTrue(written["ok"])

    def test_capture_reports_failures_in_status_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            status = os.path.join(tmp, "screen.json")

            code, data = windows_screenshot.capture_screenshot(
                os.path.join(tmp, "screen.png"),
                status_path=status,
                image_grab=FailingImageGrab,
            )

            self.assertEqual(code, 2)
            self.assertFalse(data["ok"])
            self.assertIn("screen grab failed", data["error"])
            written = json.loads(Path(status).read_text())
            self.assertFalse(written["ok"])


if __name__ == "__main__":
    unittest.main()
