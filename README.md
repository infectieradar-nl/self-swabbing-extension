## Config variables

### General

- `LOG_LEVEL`
  - configuring log level for the service
  - expected values: `debug` / `info` / `warning` / `error`
- `GIN_DEBUG_MODE`
  - use debug or release mode for the gin framework
  - expected values: `true` / `false`
- `INSTANCE_ID`
  - name of the instance the self-swabbing service will be available
  - expected value: instanceID as string, e.g. `default` / `infectieradar`

### Server

- `SELF_SWABBING_EXT_LISTEN_PORT`
  - Port number the HTTP server will listen on
  - expected value: port number, e.g., 5015
- `CORS_ALLOW_ORIGINS`
  - list of allowed origins, comma separated
- `API_KEYS`
  - comman separated list of allowed api keys. Protected endpoints will check if the HTTP header contains any of the listed keys.

- `ALLOW_ENTRY_CODE_UPLOAD`
  - toggle if the endpoint to upload new entry codes is attached or not. When not attached, the attempt to upload new codes will return 404 status.
  - expected values: `true` / `false`

### DB config

- `SELF_SWABBING_EXT_DB_CONNECTION_STR`
- `SELF_SWABBING_EXT_DB_USERNAME`
- `SELF_SWABBING_EXT_DB_PASSWORD`
- `SELF_SWABBING_EXT_DB_CONNECTION_PREFIX`

- `DB_TIMEOUT`
- `DB_IDLE_CONN_TIMEOUT`
- `DB_MAX_POOL_SIZE`
- `DB_DB_NAME_PREFIX`

### Sampler

- `SAMPLE_FILE_PATH`
  - path on the filesystem, where the "sample" CSV file is located (inlcuding the filename). This file contains samples about submission times in a typical interval and will be used to sample those times randomly.
- `TARGET_SAMPLE_COUNT`
  - target number of how many samples should be created in the interval. The sampler will open slots up to this number based on the sample file's random sampling.
  - expected value: number, e.g., `200`
- `OPEN_SLOTS_AT_INTERVAL_START`
  - number of open slots at the very beginning of the interval. This can be used as an offset to ensure particpants at the start of the interval also have chance to be included.
  - expected value: number, e.g., `5`
