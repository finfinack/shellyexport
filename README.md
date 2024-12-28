# ShellyExport

Tool to read power consumption or production measured by Shelly devices from the Shelly cloud and exporting it as either a CSV or to a Google Sheet.

## Config

```json
{
  "server": "https://shelly-145-eu.shelly.cloud",
  "auth_key": "<shelly auth key>",
  "devices": [
    {
      "id": "<shelly device ID>",
      "name": "Home",
      "type": "em-3p",
      "disabled": true,
      "google_sheet": {
        "sheet_id": "<google sheet ID>"
      }
    },
    {
      "id": "<shelly device ID>",
      "name": "Solar",
      "type": "em-1",
      "disabled": false,
      "google_sheet": {
        "sheet_id": "<google sheet ID>"
      }
    }
  ],
  "timeframe": {
    "lookback_days": 31
  },
  "google_sheet": {
    "service_account_key": "<base64 encoded service account key>",
    "spreadsheet_id": "<google spreadsheet ID>"
  }
}
```

**Shelly**

* `user_agent`: Optional field to set an HTTP User Agent different from the Go default when talking to the Shelly API.

* `auth_key` and `server`: Auth key and server required to talk to the Shelly API. Get yours by going to the [user settings](https://control.shelly.cloud/#/settings/user) and click "Get key" under "Authorization cloud key". This also reveals the `server` to talk to.

* Device `id`: The ID of the device to export. This can be seen in the [control app or Web UI](https://control.shelly.cloud/) when clicking on the device you would like to export in the settings under "Device information" (called "Device ID" as a 12 digit hex number, e.g. "aabbccddeeff").

**Google Spreadsheet**

* `service_account_key`: In order to write to a Google Sheet, you need to create a service account and a [service account key](https://cloud.google.com/iam/docs/keys-create-delete#creating) in a Google Cloud project with the Google Sheets API enabled. Export the service account key as a JSON and encode it as a base64 string (`base64 -i /path/of/key.json`).

* `spreadsheet_id`: The ID of the Google Sheet that you would like to write to. This sheet needs to be shared (editor permissions) with the service account you got the key for above. The ID is part of the URL:

  E.g. for https://docs.google.com/spreadsheets/d/1p-lTV5WPMKVi8VZ_GfGrRTwsfRjHyD6vUVSzpR7RRhA/edit?gid=0#gid=0, `1p-lTV5WPMKVi8VZ_GfGrRTwsfRjHyD6vUVSzpR7RRhA1` is the ID.

* `sheet_id`: The ID of the sheet inside the Google Sheet, i.e. which tab to write to. The tab name should suffice.

Note: Google Sheet configs can be made globally or locally for each device. At least the sheet ID has to be specific to a device though.
