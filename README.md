### RUN

Running the protofake without configuration. You can use the provided Docker image to run the protofake. The following
command will run the protofake server with the default configuration:

```bash
docker run -it --rm -p 5675:5675 --name protofake \ 
  -v /path/to/data:/data \
default23/protofake:v1.0.0
```

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

### Data dir

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

### Descriptors

TBD

### Mappings

TBD

#### Value Getters

As the response value, you can use the value getters. The value getters are predefined string prefixes that will be
replaced with the
corresponding value. The following value getters are available:

| Prefix                        | Description                                                                                                                                                                                                                                                                                            |
|-------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| $req.body.<property_name>     | The value of the request body property with the name `<property_name>`. The <property_name> is the json path to target value. For example `$req.body.resource.name` will return the value of the `name` property in the `resource` object from the request body.                                       |
| $req.metadata.<property_name> | The value of the request metadata property with the name `<property_name>`. The <property_name> is the metadata key. For example `$req.metadata.x-foo` will return the value of the `x-foo` metadata key from the request. The metadata could be an array of values, so it will joined with ` `(space) |

### Troubleshooting

Got an error on response mapping

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