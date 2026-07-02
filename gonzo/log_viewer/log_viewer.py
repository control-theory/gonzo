"""
Raw Log Viewer module.

This module provides a simple Raw Log Viewer.
"""

from typing import List
from gonzo.log_viewer.columns import ColumnCustomizer
import pandas as pd

class LogViewer:
    """
    Raw Log Viewer.

    Attributes:
        customizer (ColumnCustomizer): Customizer for column attributes.
    """

    def __init__(self):
        self.customizer = ColumnCustomizer()

    def display_logs(self, logs: List[Dict]):
        """
        Display logs.

        Args:
            logs (List[Dict]): List of log entries.
        """
        # Create a pandas DataFrame from the log data
        df = pd.DataFrame(logs)

        # Get the selected column attributes
        columns = self.customizer.get_selected_columns()

        # Display the logs using the selected column attributes
        print(df[columns].to_string(index=False))

# Example usage:
log_viewer = LogViewer()
log_viewer.display_logs([{"namespace": "default", "podName": "pod-1"}, {"namespace": "default", "podName": "pod-2"}])