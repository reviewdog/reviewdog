# Bitbucket OpenAPI

Bitbucket API swagger definitions can be found here [Bitbucket API](https://developer.atlassian.com/bitbucket/api/2/reference/meta/serialization#oai).

## Problems with official OpenAPI definitions

However, unfortunately it is not valid OpenAPI definition, see this issues:

1. [ktrysmt/go-bitbucket Issue #96](https://github.com/ktrysmt/go-bitbucket/issues/96)

1. [JIRA Bitbucket Cloud/BCLOUD-17601 - api.bitbucket.org/swagger.json is broken](https://jira.atlassian.com/browse/BCLOUD-17601)

However, I was able to generate some models via [OpenAPI Generator CLI](https://github.com/OpenAPITools/openapi-generator-cli/) with `--skip-validate-spec` ignoring the errors.
Here I copied only models related to [Code Insights](https://support.atlassian.com/bitbucket-cloud/docs/code-insights/).
Feel free to extend if needed.

Also, to make it work, in generated code URLs `{workspace}` need to be replaced with `{username}`,
because BitBucket OpenAPI definition is not correct :shrug::

**Resource URL** is `/2.0/repositories/{workspace}/{repo_slug}/commit/{commit}/reports/{reportId}`

but **Path parameters** are `username`, `repo_slug`, `commit`, `reportId`

## Attempts to generate code

- Go Swagger:

    ```sh
    $ swagger validate https://bitbucket.org/api/swagger.json
    json: cannot unmarshal bool into Go struct field SwaggerProps.definitions of type []string
    ```

- OpenAPI Generator CLI:

    ```sh
    $ docker run --rm -v "${PWD}:/local" \
        openapitools/openapi-generator-cli generate \
             -i https://api.bitbucket.org/swagger.json
             -g go \
             -o /local/out/go
    ...
    Exception in thread "main" org.openapitools.codegen.SpecValidationException: There were issues with the specification. The option can be disabled via validateSpec (Maven/Gradle) or --skip-validate-spec (CLI).
    | Error count: 53, Warning count: 17
    ...
    ```
