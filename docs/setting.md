# Torcontroller Setting

## Overview

Torcontroller is a CLI tool for managing and controlling Tor network settings. This document provides details on configuring the `torcontroller.yml` file to customize rate limits and other settings.

## Configuration File

The main configuration file for Torcontroller is located at:`/etc/torcontroller/torcontroller.yml`

### Example Configuration

Below is an example of the `torcontroller.yml` file:

```yaml
rate_limit:
  min_read_rate: 10000  # Minimum read speed in bytes per second
  min_write_rate: 5000  # Minimum write speed in bytes per second
```

### Configuration Details

- rate_limit: Specifies the minimum connection speed requirements.
  - min_read_rate: Defines the minimum read speed. Connections slower than this will be dropped or flagged.
  - min_write_rate: Defines the minimum write speed. Connections slower than this will be dropped or flagged.

## How to Set Up

1. Create or Edit Configuration File:

   - Ensure the directory `/etc/torcontroller` exists. If not, create it:

    ```bash
    sudo mkdir -p /etc/torcontroller
    ```

   - Open the configuration file in a text editor:

    ```bash
    sudo nano /etc/torcontroller/torcontroller.yml
    ```

   - Add or update the configuration as needed:

    ```yaml
    rate_limit:
        min_read_rate: 10000
        min_write_rate: 5000
    ```

2. Apply Changes:

    - Save the file and restart Torcontroller to apply the new settings:

    ```bash
    sudo systemctl restart torcontroller
    ```
