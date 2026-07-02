"""
Raw Log Viewer Documentation
==========================

Raw Log Viewer
--------------

The Raw Log Viewer is a simple module that displays log entries in a table format.

Customization API
-----------------

The Raw Log Viewer provides a customizable column attribute API that allows users to select which attributes are displayed in the Host/Service columns.

Column Customizer
-----------------

The column customizer is responsible for managing the selected column attributes. It provides methods for adding, selecting, and getting the selected column attributes.

Example Usage
-------------

Here is an example of how to use the column customizer:

.. code-block:: python

    customizer = ColumnCustomizer()
    customizer.add_column("namespace")
    customizer.add_column("podName")
    customizer.select_columns(["namespace", "podName"])
    print(customizer.get_selected_columns())  # Output: ["namespace", "podName"]