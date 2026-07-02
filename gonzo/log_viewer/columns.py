"""
Column attribute customization module.

This module provides a simple API for users to customize the Host/Service column attributes.
"""

from typing import List, Dict

class ColumnCustomizer:
    """
    Customizer for column attributes.

    Attributes:
        columns (List[str]): List of available column attributes.
        selected_columns (List[str]): List of selected column attributes.
    """

    def __init__(self):
        self.columns = []
        self.selected_columns = []

    def get_available_columns(self):
        """
        Get a list of available column attributes.

        Returns:
            List[str]: List of available column attributes.
        """
        return self.columns

    def select_columns(self, columns: List[str]):
        """
        Select column attributes.

        Args:
            columns (List[str]): List of column attributes to select.
        """
        self.selected_columns = columns

    def get_selected_columns(self):
        """
        Get a list of selected column attributes.

        Returns:
            List[str]: List of selected column attributes.
        """
        return self.selected_columns

    def add_column(self, column: str):
        """
        Add a column attribute.

        Args:
            column (str): Column attribute to add.
        """
        self.columns.append(column)

# Example usage:
customizer = ColumnCustomizer()
customizer.add_column("namespace")
customizer.add_column("podName")
customizer.select_columns(["namespace", "podName"])
print(customizer.get_selected_columns())  # Output: ["namespace", "podName"]