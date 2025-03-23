### Configuration

The protofake server can be configured using environment variables. The following table lists the available
configuration options:

| Name                          | Type   | Default | Description                                                                                                                                                                          |
|-------------------------------|--------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| WATCH_MAPPINGS_CHANGES        | bool   | false   | Enables the watching for filesystem events to track the changed mapping files.                                                                                                       |
| DATA_DIR                      | string | /data   | Is the directory, where protofake searches for the mapping and descriptor files.                                                                                                     |
| DESCRIPTOR_EXTENSIONS         | string | .pb     | A list of file extensions that protofake will analyze for registration. The separator for multiple extensions is `,`                                                                 |
| GRPC_HOST                     | string | 0.0.0.0 | Is the host address for the gRPC server.                                                                                                                                             |
| GRPC_PORT                     | int    | 5675    | Is the port for the gRPC server.                                                                                                                                                     |
| GRPC_SERVER_REFLECTION        | bool   | false   | Enables the gRPC reflection server.                                                                                                                                                  |
| GRPC_IGNORE_DUPLICATE_SERVICE | bool   | false   | Throws an error during application startup if the same service is registered multiple times. It may happen if you have multiple descriptor files with the same package+service name. |
| GRPC_DISCARD_UNKNOWN_FIELDS   | bool   | false   | Ignores the unknown fields when constructing the response from mapping.                                                                                                              |
| LOG_LEVEL                     | string | info    | Controls the log level. Possible values are: `debug`, `info`, `warn`, `error`.                                                                                                       |
| LOG_JSON_FORMAT               | bool   | true    | Prints the logs in JSON format.                                                                                                                                                      |

#### Data dir

The `DATA_DIR` is the directory where protofake searches for the mapping and descriptor files. The directory structure
should look like this:

- /data
    - mappings
        - delete_account.json
        - hello.json
    - descriptors
        - example_service.pb
        - accounts.pb

The protofake will recursively search for the mapping files in the `mappings` directory and the descriptor files in the
`descriptors`, so you can create subdirectories to organize your files if you want.

#### Descriptors

TBD

#### Mappings

TBD

### FAQ

got error on response mapping

```
invalid value for bytes field value: "your value"
```

It means the property of type (*wrapperspb.BytesValue) and expects the bytes.
Try to use the base64 encoded string in as the value, for example:

NOTE: base64 of "bytes value"

```json
{
  "response": {
    "body": {
      "property_name": "Ynl0ZXMgdmFsdWUK"
    }
  }
}
```