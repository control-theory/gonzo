"""
Tests for the Raw Log Viewer.
"""

import unittest
from gonzo.log_viewer.log_viewer import LogViewer
from gonzo.log_viewer.columns import ColumnCustomizer

class TestLogViewer(unittest.TestCase):
    def test_display_logs(self):
        # Create a log viewer
        log_viewer = LogViewer()

        # Create a column customizer
        customizer = ColumnCustomizer()
        customizer.add_column("namespace")
        customizer.add_column("podName")
        customizer.select_columns(["namespace", "podName"])

        # Create log data
        logs = [{"namespace": "default", "podName": "pod-1"}, {"namespace": "default", "podName": "pod-2"}]

        # Display the logs
        log_viewer.display_logs(logs)

        # Check the output
        expected_output = "namespace podName\ndefault pod-1\ndefault pod-2"
        self.assertEqual(log_viewer.display_logs(logs), expected_output)

if __name__ == "__main__":
    unittest.main()