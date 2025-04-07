### RUN

Running the protofake without configuration. You can use the provided Docker image to run the protofake. The following
command will run the protofake server with the default configuration:

```bash
docker run -it --rm -p 5675:5675 --name protofake \ 
  -v /path/to/data:/data \
default23/protofake:v1.1.1
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

Be aware, if the `WATCH_MAPPINGS_CHANGES` is set, The Protofake will replace all the registered mapping new ones. This
means that all the mappings that were registered through the API will be removed. Therefore, this option is not
recommended to be used in production. It is intended only for development and testing.

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

Mapping is a description of the parameters by which Protofake will generate the response. To determine what response
Protofake should return, it will use the following parameters:

- **Endpoint** - This is the GRPC name that will be used to call. For example,
  `Example.package.examPleservicehello`. Endpoint consists of three parts:
    - **package** - this is the name of the package in which the service is located. For example, `example.package`.
    - **service** - this is the name of the service that will be used to call. For example, `ExamPleservice`.
    - **method** - this is the name of the method that will be used to call. For example, `Hello`.
- **Metadata** - object, the key is the name of request metadata, and the value is [ValueMatcher](#value-matcher)
  object.
- **Request** - object, the key is the name of the parameter (json-path to the target parameter), and
  value is [ValueMatcher](#value-matcher) object.

If the request matches the metadata and request parameters, the response will be generated by the given mapping. The
response mapping consists of the following parameters:

- **body** - object, the key is the name of the parameter (json-path to the target parameter), and the value that should
  remain in the response. The values can be [ValueGetter](#value-getters) or a primitive value. If no value is
  specified, the default value for the given type will be returned. For example, `0` will be returned for `int32`, an
  empty string for `string`, `false` for `bool`, and so on.
- **code** - the status code that will be returned. The default value is `OK`. The code should be a valid gRPC status
  code. For example, `NOT_FOUND`, `INVALID_ARGUMENT`, `UNAUTHENTICATED`, etc.
- **error_message** - the error message that will be returned. The default value is `""`. If the code is not `OK`, the
  error message will be returned as part of the response.

> NOTE: If you want respond with the default properties and code=OK, you could omit the response mapping. The
> response will be returned with the default values for each property.

Also, the mapping has an `id` property. This is the unique identifier over the other endpoint mappings. The id is used
to
identify the mapping in the logs and in the admin API. When the mapping is applied through the API, the id is used to
identify existing mapping. If the mapping with the same id already exists, it will be replaced with the new one. The id
is not used to match the request, so you can use the same id for multiple mappings through different endpoints. This is
the optional property, if not set, the mapping will be generated with the random id.

If several mappings are registered for endpoint, which satisfy the request, then the one that is used will be used
Registered by the last. The procedure for verifying conformity to the request occurs in the manner from the latest to
the first.

The file with mapping could be a single json object or an array of json objects and have the `.json` extension. The
mapping should be like:

```json
{
  "id": "hello-postman",
  "endpoint": "/greeter.v1.Greeter/SayHello",
  "metadata": {
    "user-agent": {
      "rule": "equal",
      "value": "PostmanRuntime/*"
    }
  },
  "request_body": {
    "username": {
      "rule": "glob",
      "value": "pos*man"
    }
  },
  "response": {
    "body": {
      "greeting": "Hello Postman!",
      "username": "$req.body.username"
    }
  }
}
```

See the directory [example](./example) for more examples.

> NOTE: To indicate the parameter names, it is necessary to use the field names as they are indicated in the Proto file.
> For example, if in the Proto file the field is called `string user_id = 1;`, then in the mapping it must be indicated
> as `user_id`.

#### Value Matcher

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